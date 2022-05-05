package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

type DockerClient struct {
	Client *dockerclient.Client
}

func NewDockerClient() (*DockerClient, error) {
	client, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		Client: client,
	}, nil
}

func (dockerClient *DockerClient) IsInstalled() bool {
	_, err := dockerClient.Client.Info(context.Background())
	return err == nil
}

func (dockerClient *DockerClient) GetContainer(nameOrId string) (*types.Container, error) {

	containers, err := dockerClient.Client.ContainerList(context.Background(), types.ContainerListOptions{
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

func (dockerClient *DockerClient) RemoveContainer(nameOrId string) error {
	ctx := context.Background()

	container, err := dockerClient.GetContainer(nameOrId)
	if err != nil {
		return err
	}
	log.Debug().Msgf("Container Stop: %+v\n", container)
	timeout := time.Millisecond * 100
	err = dockerClient.Client.ContainerStop(ctx, container.ID, &timeout)
	if err != nil {
		return err
	}
	err = dockerClient.Client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		return err
	}
	return nil
}
