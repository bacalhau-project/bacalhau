package docker

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type DockerExecutor struct {
	// the global context for stopping any running jobs
	Ctx context.Context
	// the address to connect to the accompanying ipfs server
	// this is optional as we might be wanting to only run jobs
	// with other types of executors
	IpfsMultiAddress string
	// are we running in bad actor mode? (useful for tests)
	BadActor bool
}

func NewDockerExecutor(
	ctx context.Context,
	ipfsMultiAddress string,
	badActor bool,
) (*DockerExecutor, error) {
	dockerExecutor := &DockerExecutor{
		IpfsMultiAddress: ipfsMultiAddress,
		BadActor:         badActor,
		Ctx:              ctx,
	}
	return dockerExecutor, nil
}

func (docker *DockerExecutor) IsInstalled() (bool, error) {
	return false, nil
}

func (docker *DockerExecutor) PrepareStorage(storageProvider storage.Storage, volume types.StorageSpec) error {
	return nil
}

func (docker *DockerExecutor) RunJob(job *types.Job) ([]types.StorageSpec, error) {
	outputs := []types.StorageSpec{}
	return outputs, nil
}
