package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/filecoin-project/bacalhau/pkg/docker/tracing"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.ptx.dk/multierrgroup"
	"go.uber.org/multierr"
)

type Client struct {
	tracing.TracedClient
}

func NewDockerClient() (*Client, error) {
	client, err := tracing.NewTracedClient()
	if err != nil {
		return nil, err
	}
	return &Client{
		client,
	}, nil
}

func (c *Client) IsInstalled(ctx context.Context) bool {
	_, err := c.Info(ctx)
	return err == nil
}

func (c *Client) HostGatewayIP(ctx context.Context) (net.IP, error) {
	response, err := c.NetworkInspect(ctx, "bridge", types.NetworkInspectOptions{})
	if err != nil {
		return net.IP{}, err
	}
	if configs := response.IPAM.Config; len(configs) < 1 {
		return net.IP{}, fmt.Errorf("bridge network unattached")
	} else {
		return net.ParseIP(configs[0].Gateway), nil
	}
}

func (c *Client) removeContainers(ctx context.Context, filterz filters.Args) error {
	containers, err := c.ContainerList(ctx, types.ContainerListOptions{All: true, Filters: filterz})
	if err != nil {
		return err
	}

	wg := multierrgroup.Group{}
	for _, container := range containers {
		container := container
		wg.Go(func() error {
			return c.RemoveContainer(ctx, container.ID)
		})
	}
	return wg.Wait()
}

func (c *Client) removeNetworks(ctx context.Context, filterz filters.Args) error {
	networks, err := c.NetworkList(ctx, types.NetworkListOptions{Filters: filterz})
	if err != nil {
		return err
	}

	wg := multierrgroup.Group{}
	for _, network := range networks {
		network := network
		wg.Go(func() error {
			log.Ctx(ctx).Debug().Str("Network", network.ID).Msg("Network Stop")
			return c.NetworkRemove(ctx, network.ID)
		})
	}
	return wg.Wait()
}

func (c *Client) RemoveObjectsWithLabel(ctx context.Context, labelName, labelValue string) error {
	filterz := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", labelName, labelValue)),
	)

	containerErr := c.removeContainers(ctx, filterz)
	networkErr := c.removeNetworks(ctx, filterz)
	return multierr.Combine(containerErr, networkErr)
}

func (c *Client) FollowLogs(ctx context.Context, id string) (stdout, stderr io.Reader, err error) {
	cont, err := c.ContainerInspect(ctx, id)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get container")
	}

	ctx = log.Ctx(ctx).With().Str("ContainerID", cont.ID).Str("Image", cont.Image).Logger().WithContext(ctx)
	logsReader, err := c.ContainerLogs(ctx, cont.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get container logs")
	}

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	go func() {
		stdoutBuffer := bufio.NewWriter(stdoutWriter)
		stderrBuffer := bufio.NewWriter(stderrWriter)
		defer closer.CloseWithLogOnError("stderrWriter", stderrWriter)
		defer closer.CloseWithLogOnError("stdoutWriter", stdoutWriter)
		defer stderrBuffer.Flush()
		defer stdoutBuffer.Flush()
		defer closer.CloseWithLogOnError("logsReader", logsReader)

		_, err = stdcopy.StdCopy(stdoutBuffer, stderrBuffer, logsReader)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Ctx(ctx).Err(err).Msg("error reading container logs")
		}
	}()

	return stdoutReader, stderrReader, nil
}

func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	log.Ctx(ctx).Debug().Str("id", id).Msgf("Container Stop")
	timeout := time.Millisecond * 100
	if err := c.ContainerStop(ctx, id, &timeout); err != nil {
		if dockerclient.IsErrNotFound(err) {
			return nil
		}
		return errors.WithStack(err)
	}
	err := c.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (c *Client) PullImage(ctx context.Context, image string) error {
	_, _, err := c.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil
	}
	if !dockerclient.IsErrNotFound(err) {
		return err
	}

	log.Ctx(ctx).Debug().Str("image", image).Msg("Pulling image as it wasn't found")

	output, err := c.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	defer closer.CloseWithLogOnError("image-pull", output)

	stop := make(chan struct{}, 1)
	defer func() {
		stop <- struct{}{}
	}()
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()

	layers := &sync.Map{}
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				logImagePullStatus(ctx, layers)
			}
		}
	}()

	dec := json.NewDecoder(output)
	for {
		var mess jsonmessage.JSONMessage
		if err := dec.Decode(&mess); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if mess.Aux != nil {
			continue
		}
		if mess.Error != nil {
			return mess.Error
		}
		layers.Store(mess.ID, mess)
	}
}

func logImagePullStatus(ctx context.Context, m *sync.Map) {
	withUnits := map[string]*zerolog.Event{}
	withoutUnits := map[string][]string{}
	m.Range(func(_, value any) bool {
		mess := value.(jsonmessage.JSONMessage)

		if mess.Progress == nil || mess.Progress.Current <= 0 {
			withoutUnits[mess.Status] = append(withoutUnits[mess.Status], mess.ID)
		} else {
			var status string
			if mess.Progress.Total <= 0 {
				status = fmt.Sprintf("%d %s", mess.Progress.Total, mess.Progress.Units)
			} else {
				status = fmt.Sprintf("%.3f%%", float64(mess.Progress.Current)/float64(mess.Progress.Total)*100) //nolint:gomnd
			}

			if _, ok := withUnits[mess.Status]; !ok {
				withUnits[mess.Status] = zerolog.Dict()
			}

			withUnits[mess.Status].Str(mess.ID, status)
		}

		return true
	})
	e := log.Ctx(ctx).Debug()
	for s, l := range withUnits {
		e = e.Dict(s, l)
	}
	for s, l := range withoutUnits {
		sort.Strings(l)
		e = e.Strs(s, l)
	}

	e.Msg("Pulling layers")
}
