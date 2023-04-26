package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/docker/tracing"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.ptx.dk/multierrgroup"
	"go.uber.org/multierr"
	"golang.org/x/exp/slices"
)

const ImagePullError = `Could not pull image %q - could be due to repo/image not existing, ` +
	`or registry needing authorization`

const DistributionInspectError = `Could not inspect image %q - could be due to repo/image not existing, ` +
	`or registry needing authorization`

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

func (c *Client) FindContainer(ctx context.Context, label string, value string) (string, error) {
	containers, err := c.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return "", err
	}

	for _, ctr := range containers {
		if ctr.Labels[label] == value {
			return ctr.ID, nil
		}
	}

	return "", fmt.Errorf("unable to find container for %s=%s", label, value)
}

func (c *Client) FollowLogs(ctx context.Context, id string) (stdout, stderr io.Reader, err error) {
	cont, err := c.ContainerInspect(ctx, id)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get container")
	}

	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}

	ctx = log.Ctx(ctx).With().Str("ContainerID", cont.ID).Str("Image", cont.Image).Logger().WithContext(ctx)
	logsReader, err := c.ContainerLogs(ctx, cont.ID, logOptions)
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

func (c *Client) GetOutputStream(ctx context.Context, id string, since string, follow bool) (io.ReadCloser, error) {
	cont, err := c.ContainerInspect(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get container")
	}

	if !cont.State.Running {
		return nil, errors.Wrap(err, "cannot get logs when container is not running")
	}

	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
	}
	if since != "" {
		logOptions.Since = since
	}

	ctx = log.Ctx(ctx).With().Str("ContainerID", cont.ID).Str("Image", cont.Image).Logger().WithContext(ctx)
	logsReader, err := c.ContainerLogs(ctx, cont.ID, logOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get container logs")
	}

	return logsReader, nil
}

func (c *Client) RemoveContainer(ctx context.Context, id string) error {
	log.Ctx(ctx).Debug().Str("id", id).Msgf("Container Stop")
	timeout := time.Millisecond * 100
	if err := c.ContainerStop(ctx, id, timeout); err != nil {
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

// ImagePlatforms will retrieve the manifest describing the platforms that are
// usable by the provided image. This currently retrieves the information via
// a call to ImageDistribution which provides the same information.
func (c *Client) ImagePlatforms(ctx context.Context, image string, dockerCreds config.DockerCredentials) ([]v1.Platform, error) {
	manifest, err := c.ImageDistribution(ctx, image, dockerCreds)
	if err != nil {
		return nil, err
	}

	return manifest.platforms, nil
}

func (c *Client) SupportedPlatforms(ctx context.Context) ([]v1.Platform, error) {
	version, err := c.ServerVersion(ctx)
	if err != nil {
		return nil, err
	}

	engineIdx := slices.IndexFunc(version.Components, func(v types.ComponentVersion) bool {
		return v.Name == "Engine"
	})

	// Note that 'Os' is linux on Darwin/Windows platforms that are running Linux VMs
	engine := version.Components[engineIdx].Details
	return []v1.Platform{
		{
			Architecture: engine["Arch"],
			OS:           engine["Os"],
		},
	}, nil
}

// ImageDistribution fetches the details for the specified image by asking
// docker to fetch the distribution manifest from the registry. The digest
// retrieved may not appear accurate when compared to the hub website but
// this is expected.
//
// cf:
//   - https://github.com/moby/moby/issues/40636)
//   - https://github.com/docker/roadmap/issues/262
//
// When a docker image is available on only a single platform, the digest
// shown will be the digest pointing directly at the manifest for that image
// on that platform (as shown by the docker hub).  Where multiple platforms
// are available, the digest is pointing to a top level document describing
// all of the different platform manifests available.
//
// In either case, `docker pull` will do the correct thing and download the
// image for your platform. For example:
//
// $ docker manifest inspect  bitnami/rabbitmq@sha256:0be0d2a2 ...
//
//	"manifests": [ {
//		  "digest": "sha256:959a02013e8ab5538167f9....",
//		  "platform": { "architecture": "amd64", "os": "linux" }
//		},
//		{
//		  "digest": "sha256:11ee2c7e9e69e3a8311a19....",
//		  "platform": { "architecture": "arm64", "os": "linux"}
//		}]
//
// $ docker pull bitnami/rabbitmq@sha256:0be0d2a2 ...
// $ docker image ls
// bitnami/rabbitmq ... 48603925e10c
//
// The digest 486039 can be found in manifest sha256:11ee2c7e which is the manifest for
// the current authors machine.
//
// $ docker manifest inspect bitnami/rabbitmq@sha256:11ee2c7e
//
//	  "config": {
//		   "size": 7383,
//		   "digest": "sha256:48603925e10c01936ea4258f...."
//	  }
//
// This is the image that will finally be installed.
func (c *Client) ImageDistribution(ctx context.Context, image string, dockerCreds config.DockerCredentials) (*ImageManifest, error) {
	authToken := getAuthToken(ctx, image, dockerCreds)

	dist, err := c.DistributionInspect(ctx, image, authToken)
	if err != nil {
		return nil, err
	}

	manifest := &ImageManifest{
		digest: dist.Descriptor.Digest.Encoded(),
	}
	copy(manifest.platforms, dist.Platforms)
	return manifest, nil
}

func (c *Client) PullImage(ctx context.Context, image string, dockerCreds config.DockerCredentials) error {
	_, _, err := c.ImageInspectWithRaw(ctx, image)
	if err == nil {
		// If there is no error, then return immediately as it means we have the docker image
		// being discussed. No need to pull it.
		return nil
	}

	if !dockerclient.IsErrNotFound(err) {
		// The only error we wanted to see was a not found error which means we don't have
		// the image being requested.
		return err
	}

	log.Ctx(ctx).Debug().Str("image", image).Msg("Pulling image as it wasn't found")

	pullOptions := types.ImagePullOptions{
		RegistryAuth: getAuthToken(ctx, image, dockerCreds),
	}

	output, err := c.ImagePull(ctx, image, pullOptions)
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

func getAuthToken(ctx context.Context, image string, dockerCreds config.DockerCredentials) string {
	if dockerCreds.IsValid() {
		// We only currently support auth for the default registry, so any
		// pulls for `image` or `user/image` should be okay, anything trying
		// to pull `repo/user/image` should not.
		if strings.Count(image, "/") < 2 {
			authConfig := types.AuthConfig{
				Username: dockerCreds.Username,
				Password: dockerCreds.Password,
			}

			encodedJSON, err := json.Marshal(authConfig)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to encode docker credentials")
			} else {
				log.Ctx(ctx).Info().Msg("authenticated inspect from docker registry")
				return base64.URLEncoding.EncodeToString(encodedJSON)
			}
		} else {
			log.Ctx(ctx).Info().Msg("cannot authenticate for custom registry")
		}
	}

	return ""
}
