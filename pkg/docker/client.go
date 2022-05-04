package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"
)

type DockerClient struct {
	Ctx    context.Context
	Client *dockerclient.Client
}

func NewDockerClient(
	ctx context.Context,
) (*DockerClient, error) {
	client, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		Ctx:    ctx,
		Client: client,
	}, nil
}

func (dockerClient *DockerClient) IsInstalled() bool {
	_, err := dockerClient.Client.Info(dockerClient.Ctx)
	return err == nil
}

func (dockerClient *DockerClient) GetContainer(nameOrId string) (*types.Container, error) {
	containers, err := dockerClient.Client.ContainerList(dockerClient.Ctx, types.ContainerListOptions{
		// we want to know about stopped containers too
		All: true,
	})
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.ID == nameOrId {
			return &container, nil
		}

		for _, containerName := range container.Names {
			if containerName == nameOrId {
				return &container, nil
			}
		}
	}
	return nil, nil
}

func (dockerClient *DockerClient) RemoveContainer(id string) error {
	err := dockerClient.Client.ContainerStop(dockerClient.Ctx, id, nil)
	if err != nil {
		return err
	}
	err = dockerClient.Client.ContainerRemove(dockerClient.Ctx, id, types.ContainerRemoveOptions{
		Force: true,
	})
	if err != nil {
		return err
	}
	return nil
}
