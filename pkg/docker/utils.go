package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/moby/moby/pkg/stdcopy"
	"github.com/rs/zerolog/log"
)

func NewDockerClient() (*dockerclient.Client, error) {
	return dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
}

func IsInstalled(dockerClient *dockerclient.Client) bool {
	_, err := dockerClient.Info(context.Background())
	return err == nil
}

func GetContainer(dockerClient *dockerclient.Client, nameOrID string) (*types.Container, error) {
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})

	if err != nil {
		return nil, err
	}
	// TODO: #287 Fix if when we care about optimization of memory (224 bytes copied per loop)
	// nolint:gocritic // will fix when we care
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

func GetContainersWithLabel(dockerClient *dockerclient.Client, labelName, labelValue string) ([]types.Container, error) {
	results := []types.Container{}
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})

	if err != nil {
		return nil, err
	}
	// TODO: #287 Fix if when we care about optimization of memory (224 bytes copied per loop)
	// nolint:gocritic // will fix when we care
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

func GetLogs(dockerClient *dockerclient.Client, nameOrID string) (stdout, stderr string, err error) {
	container, err := GetContainer(dockerClient, nameOrID)
	if err != nil {
		return "", "", err
	}
	if container == nil {
		return "", "", fmt.Errorf("no container found: %s", nameOrID)
	}
	logsReader, err := dockerClient.ContainerLogs(context.Background(), container.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", "", err
	}
	stdoutBuffer := bytes.NewBuffer([]byte{})
	stderrBuffer := bytes.NewBuffer([]byte{})
	_, err = stdcopy.StdCopy(stdoutBuffer, stderrBuffer, logsReader)
	if err != nil {
		return "", "", err
	}
	return stdoutBuffer.String(), stderrBuffer.String(), nil
}

func RemoveContainer(dockerClient *dockerclient.Client, nameOrID string) error {
	ctx := context.Background()

	container, err := GetContainer(dockerClient, nameOrID)
	if err != nil {
		return err
	}
	if container == nil {
		return nil
	}
	log.Debug().Msgf("Container Stop: %s", container.ID)
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

func WaitForContainer(client *dockerclient.Client, id string, maxAttempts int, delay time.Duration) error {
	waiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Handler: func() (bool, error) {
			container, err := GetContainer(client, id)
			if err != nil {
				return false, err
			}
			if container == nil {
				return false, nil
			}
			return container.State == "running", nil
		},
	}
	return waiter.Wait()
}

func WaitForContainerLogs(client *dockerclient.Client, id string, maxAttempts int, delay time.Duration, findString string) (string, error) {
	lastLogs := ""
	waiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Handler: func() (bool, error) {
			container, err := GetContainer(client, id)
			if err != nil {
				return false, err
			}
			if container == nil {
				return false, nil
			}
			if container.State != "running" {
				return false, nil
			}
			stdout, stderr, err := GetLogs(client, id)
			if err != nil {
				return false, err
			}
			lastLogs = stdout + "\n" + stderr
			return strings.Contains(stdout, findString) || strings.Contains(stderr, findString), nil
		},
	}
	err := waiter.Wait()
	return lastLogs, err
}

func PullImage(dockerClient *dockerclient.Client, image string) error {
	imagePullStream, err := dockerClient.ImagePull(
		context.Background(),
		image,
		types.ImagePullOptions{},
	)

	if err != nil {
		return err
	}

	if config.IsDebug() {
		_, err = io.Copy(os.Stdout, imagePullStream)
		if err != nil {
			return err
		}
	}

	return imagePullStream.Close()
}
