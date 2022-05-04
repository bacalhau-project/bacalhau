package docker

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type DockerExecutor struct {
	// the global context for stopping any running jobs
	Ctx context.Context

	// the storage providers we can implement for a job
	StorageProviders map[string]storage.StorageProvider

	Client *docker.DockerClient
}

func NewDockerExecutor(
	ctx context.Context,
	storageProviders map[string]storage.StorageProvider,
) (*DockerExecutor, error) {
	dockerClient, err := docker.NewDockerClient(ctx)
	if err != nil {
		return nil, err
	}
	dockerExecutor := &DockerExecutor{
		Ctx:              ctx,
		StorageProviders: storageProviders,
		Client:           dockerClient,
	}
	return dockerExecutor, nil
}

func (docker *DockerExecutor) getStorageProvider(engine string) (storage.StorageProvider, error) {
	return executor.GetStorageProvider(engine, docker.StorageProviders)
}

// check if docker itself is installed
func (docker *DockerExecutor) IsInstalled() (bool, error) {
	isRunning := docker.Client.IsInstalled()
	return isRunning, nil
}

func (docker *DockerExecutor) HasStorage(volume types.StorageSpec) (bool, error) {
	storage, err := docker.getStorageProvider(volume.Engine)
	if err != nil {
		return false, err
	}
	return storage.HasStorage(volume)
}

func (docker *DockerExecutor) RunJob(job *types.Job) ([]types.StorageSpec, error) {
	outputs := []types.StorageSpec{}
	// loop over the job storage inputs and prepare them
	for _, input := range job.Spec.Inputs {
		storage, err := docker.getStorageProvider(input.Engine)
		if err != nil {
			return outputs, err
		}
		volumeMount, err := storage.PrepareStorage(input)
		if err != nil {
			return outputs, err
		}
		if volumeMount == nil {
			return outputs, fmt.Errorf("no volume mount was returned for input: %+v\n", input)
		}
		fmt.Printf("\n\n\nMounted %s to %s\n\n\n", volumeMount.Source, volumeMount.Target)
	}
	return outputs, nil
}
