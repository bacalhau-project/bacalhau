package ipfs

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type DockerStorageIPFS struct {
	Ctx              context.Context
	IpfsMultiAddress string
}

func NewDockerStorageIPFS(
	ctx context.Context,
	ipfsMultiAddress string,
) (*DockerStorageIPFS, error) {
	dockerStorageIPFS := &DockerStorageIPFS{
		Ctx:              ctx,
		IpfsMultiAddress: ipfsMultiAddress,
	}
	return dockerStorageIPFS, nil
}

func (docker *DockerStorageIPFS) IsInstalled() (bool, error) {
	return false, nil
}

func (docker *DockerStorageIPFS) HasStorage(volume types.StorageSpec) error {
	return nil
}

func (docker *DockerStorageIPFS) PrepareStorage(volume types.StorageSpec) (*storage.StorageVolume, error) {
	return nil, nil
}
