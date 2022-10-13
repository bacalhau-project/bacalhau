package devstack

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/nat"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

const defaultImage = "ghcr.io/bacalhau-project/lotus-filecoin-image:v0.0.1"

type LotusNode struct {
	client    *dockerclient.Client
	image     string
	container string

	// Port is the port number that Lotus will be listening on
	Port string
	// Dir is the directory where files to be uploaded to Lotus should be stored
	Dir string
	// Token is actor in the local network with some FIL to do some work
	Token string
}

func newLotusNode(ctx context.Context) (*LotusNode, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/devstack.newlotusnode")
	defer span.End()

	image := defaultImage
	if e, ok := os.LookupEnv("LOTUS_TEST_IMAGE"); ok {
		image = e
	}

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil, err
	}

	if err := pullImage(ctx, dockerClient, image); err != nil {
		closer.CloseWithLogOnError("docker", dockerClient)
		return nil, err
	}

	return &LotusNode{
		client: dockerClient,
		image:  image,
	}, nil
}

// start performs the work of actually starting the Lotus container. This is separated from the constructor so the user
// can cancel and still have the container, which may not be healthy yet, cleaned up via Close.
func (l *LotusNode) start(ctx context.Context) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/devstack.start")
	defer span.End()

	dir, err := ioutil.TempDir("", "bacalhau-lotus")
	if err != nil {
		return err
	}

	l.Dir = dir

	c, err := l.client.ContainerCreate(ctx, &container.Config{
		Image: l.image,
	}, &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			"1234/tcp": {{}},
		},
		Mounts: []mount.Mount{
			// Mount the temp directory at the same place within the container to aviod confusion between paths outside the
			// container, that the user sees, and paths within the container, that the ClientImport command uses.
			{
				Type:     mount.TypeBind,
				ReadOnly: true,
				Source:   dir,
				Target:   dir,
			},
		},
	}, nil, nil, "")
	if err != nil {
		return err
	}

	l.container = c.ID

	log.Debug().
		Str("image", l.image).
		Str("dir", l.Dir).
		Str("containerId", l.container).
		Msg("Starting Lotus container")

	if err := l.client.ContainerStart(ctx, l.container, dockertypes.ContainerStartOptions{}); err != nil {
		return err
	}

	if err := l.waitForLotusToBeHealthy(ctx); err != nil {
		if err := l.Close(); err != nil { //nolint:govet
			log.Err(err).Msgf(`Problem occurred when giving up waiting for Lotus to become healthy`)
		}
		return err
	}

	return nil
}

func (l *LotusNode) waitForLotusToBeHealthy(ctx context.Context) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/devstack.waitforlotustobehealthy")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute) //nolint:gomnd
	defer cancel()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		state, err := l.client.ContainerInspect(ctx, l.container)
		if err != nil {
			return err
		}

		if state.State.Health.Status == dockertypes.Healthy {
			l.Port = state.NetworkSettings.Ports["1234/tcp"][0].HostPort
			break
		}

		e := log.Debug()
		if len(state.State.Health.Log) != 0 {
			e = e.Str("last-health-check", strings.TrimSpace(state.State.Health.Log[len(state.State.Health.Log)-1].Output))
		}
		e.Msg("Lotus not healthy yet")
		time.Sleep(5 * time.Second) //nolint:gomnd
	}

	content, _, err := l.client.CopyFromContainer(ctx, l.container, "/home/lotus_user/.lotus-local-net/token")
	if err != nil {
		return err
	}

	defer closer.CloseWithLogOnError("content", content)

	tarContent := tar.NewReader(content)
	if _, err := tarContent.Next(); err != nil { //nolint:govet
		return err
	}

	token, err := ioutil.ReadAll(tarContent)
	if err != nil {
		return err
	}

	l.Token = string(token)

	return nil
}

func (l *LotusNode) Close() error {
	defer closer.CloseWithLogOnError("Docker client", l.client)
	if l.container != "" {
		if err := docker.RemoveContainer(context.Background(), l.client, l.container); err != nil {
			return err
		}
	}

	if l.Dir != "" {
		// This may not happen if Docker fails to remove the container, but this isn't seen as a big problem
		// as the OS is expected to clean it up
		if err := os.RemoveAll(l.Dir); err != nil {
			return err
		}
	}

	return nil
}

func pullImage(ctx context.Context, client dockerclient.ImageAPIClient, image string) error {
	_, _, err := client.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil
	}
	if !dockerclient.IsErrNotFound(err) {
		return err
	}

	log.Debug().Str("image", image).Msg("Pulling image as it wasn't found")

	output, err := client.ImagePull(ctx, image, dockertypes.ImagePullOptions{})
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
				logImagePullStatus(layers)
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

func logImagePullStatus(m *sync.Map) {
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
	e := log.Debug()
	for s, l := range withUnits {
		e = e.Dict(s, l)
	}
	for s, l := range withoutUnits {
		sort.Strings(l)
		e = e.Strs(s, l)
	}

	e.Msg("Pulling layers")
}
