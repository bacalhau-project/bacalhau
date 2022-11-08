package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ErrContainerMarkedForRemoval indicates that the docker daemon is about to
// delete, or has already deleted, the given container.
var ErrContainerMarkedForRemoval = fmt.Errorf("docker container marked for removal")

func NewDockerClient() (*dockerclient.Client, error) {
	return dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
}

func IsInstalled(ctx context.Context, dockerClient *dockerclient.Client) bool {
	_, err := dockerClient.Info(ctx)
	return err == nil
}

func GetContainer(ctx context.Context, dockerClient *dockerclient.Client, nameOrID string) (*types.Container, error) {
	containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	// TODO: #287 Fix if when we care about optimization of memory (224 bytes copied per loop)
	//nolint:gocritic // will fix when we care
	for _, container := range containers {
		if container.ID == nameOrID {
			return &container, nil
		}

		if container.ID[0:11] == nameOrID {
			return &container, nil
		}

		for _, containerName := range container.Names {
			if containerName == fmt.Sprintf("/%s", nameOrID) {
				return &container, nil
			}
		}
	}

	return nil, nil
}

func GetContainersWithLabel(ctx context.Context,
	dockerClient *dockerclient.Client,
	labelName, labelValue string) ([]types.Container, error) {
	results := []types.Container{}
	containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})

	if err != nil {
		return nil, err
	}
	// TODO: #287 Fix if when we care about optimization of memory (224 bytes copied per loop)
	//nolint:gocritic // will fix when we care
	for _, container := range containers {
		value, ok := container.Labels[labelName]
		if !ok {
			continue
		}
		if value == labelValue {
			results = append(results, container)
		}
	}
	return results, nil
}

func GetLogs(ctx context.Context, dockerClient *dockerclient.Client, nameOrID string) (stdout, stderr string, err error) {
	container, err := GetContainer(ctx, dockerClient, nameOrID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get container: %w", err)
	}
	if container == nil {
		return "", "", fmt.Errorf("no container found: %s", nameOrID)
	}

	logsReader, err := dockerClient.ContainerLogs(ctx, container.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		// String checking is unfortunately the best we have, as errors are
		// returned by the docker server as strings, and aren't strongly typed.
		if strings.Contains(err.Error(), "can not get logs from container which is dead or marked for removal") {
			return "", "", ErrContainerMarkedForRemoval
		}

		return "", "", fmt.Errorf("failed to get container logs: %w", err)
	}

	stdoutBuffer := bytes.NewBuffer([]byte{})
	stderrBuffer := bytes.NewBuffer([]byte{})
	_, err = stdcopy.StdCopy(stdoutBuffer, stderrBuffer, logsReader)
	if err != nil {
		return "", "", err
	}

	return stdoutBuffer.String(), stderrBuffer.String(), nil
}

func RemoveContainer(ctx context.Context, dockerClient *dockerclient.Client, nameOrID string) error {
	container, err := GetContainer(ctx, dockerClient, nameOrID)
	if err != nil {
		return err
	}
	if container == nil {
		return nil
	}
	log.Ctx(ctx).Debug().Msgf("Container Stop: %s", container.ID)
	timeout := time.Millisecond * 100
	err = dockerClient.ContainerStop(ctx, container.ID, &timeout)
	if err != nil {
		return err
	}
	err = dockerClient.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		return err
	}
	return nil
}

func WaitForContainerLogs(ctx context.Context,
	client *dockerclient.Client,
	id string,
	maxAttempts int,
	delay time.Duration,
	findString string) (string, error) {
	lastLogs := ""
	waiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Handler: func() (bool, error) {
			container, err := GetContainer(ctx, client, id)
			if err != nil {
				return false, err
			}
			if container == nil {
				return false, nil
			}
			if container.State != "running" {
				return false, nil
			}
			stdout, stderr, err := GetLogs(ctx, client, id)
			if err != nil {
				return false, err
			}
			lastLogs = stdout + "\n" + stderr
			return strings.Contains(stdout, findString) || strings.Contains(stderr, findString), nil
		},
	}
	err := waiter.Wait(ctx)
	return lastLogs, err
}

func PullImage(ctx context.Context, dockerClient *dockerclient.Client, image string) error {
	_, _, err := dockerClient.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil
	}
	if !dockerclient.IsErrNotFound(err) {
		return err
	}

	log.Debug().Str("image", image).Msg("Pulling image as it wasn't found")

	output, err := dockerClient.ImagePull(ctx, image, types.ImagePullOptions{})
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
