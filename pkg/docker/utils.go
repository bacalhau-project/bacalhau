package docker

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

func NewDockerClient() (*dockerclient.Client, error) {
	return dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
}

func IsInstalled(dockerClient *dockerclient.Client) bool {
	_, err := dockerClient.Info(context.Background())
	return err == nil
}

func GetContainer(dockerClient *dockerclient.Client, nameOrId string) (*types.Container, error) {

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})

	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.ID == nameOrId {
			return &container, nil
		}

		if container.ID[0:11] == nameOrId {
			return &container, nil
		}

		for _, containerName := range container.Names {
			if containerName == fmt.Sprintf("/%s", nameOrId) {
				return &container, nil
			}
		}
	}
	return nil, nil
}

func GetLogsWithOptions(dockerClient *dockerclient.Client, nameOrId string, options types.ContainerLogsOptions) (string, error) {
	container, err := GetContainer(dockerClient, nameOrId)
	if err != nil {
		return "", err
	}
	if container == nil {
		return "", fmt.Errorf("No container found: %s", nameOrId)
	}
	logsReader, err := dockerClient.ContainerLogs(context.Background(), container.ID, options)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(logsReader)
	return buf.String(), nil
}

func GetLogs(dockerClient *dockerclient.Client, nameOrId string) (string, error) {
	return GetLogsWithOptions(dockerClient, nameOrId, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
}

func RemoveContainer(dockerClient *dockerclient.Client, nameOrId string) error {
	ctx := context.Background()

	container, err := GetContainer(dockerClient, nameOrId)
	if err != nil {
		return err
	}
	log.Debug().Msgf("Container Stop: %+v\n", container)
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
		Logging:     true,
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

func WaitForContainerLogs(client *dockerclient.Client, id string, maxAttempts int, delay time.Duration, findString string) error {
	waiter := &system.FunctionWaiter{
		Name:        fmt.Sprintf("wait for container to be running: %s", id),
		MaxAttempts: maxAttempts,
		Delay:       delay,
		Logging:     true,
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
			logs, err := GetLogs(client, id)
			if err != nil {
				return false, err
			}
			return strings.Contains(logs, findString), nil
		},
	}
	return waiter.Wait()
}
