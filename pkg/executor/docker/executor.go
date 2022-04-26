package docker

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/types"
)

type DockerExecutor struct {
	// the address to connect to the accompanying ipfs server
	IpfsMultiAddress string
	// are we running in bad actor mode? (useful for tests)
	BadActor bool
	// the global context for stopping any running jobs
	Ctx context.Context
}

func NewDockerExecutor(
	ipfsMultiAddress string,
	badActor bool,
	ctx context.Context,
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

func (docker *DockerExecutor) HasStorage(storage types.JobStorage) (bool, error) {
	return false, nil
}

func (docker *DockerExecutor) PrepareStorage(storage types.JobStorage) error {
	return nil
}

func (docker *DockerExecutor) RunJob(job *types.Job) error {
	return nil
}
