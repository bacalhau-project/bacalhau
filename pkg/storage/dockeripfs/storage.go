package dockeripfs

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type StorageDockerIPFS struct {
	Ctx              context.Context
	IpfsMultiAddress string
}

func NewStorageDockerIPFS(
	ctx context.Context,
	ipfsMultiAddress string,
) (*StorageDockerIPFS, error) {
	StorageDockerIPFS := &StorageDockerIPFS{
		Ctx:              ctx,
		IpfsMultiAddress: ipfsMultiAddress,
	}
	return StorageDockerIPFS, nil
}

func (docker *StorageDockerIPFS) IsInstalled() (bool, error) {
	return true, nil
}

func (docker *StorageDockerIPFS) HasStorage(volume types.StorageSpec) (bool, error) {
	return true, nil
}

func (docker *StorageDockerIPFS) PrepareStorage(volume types.StorageSpec) (*storage.StorageVolume, error) {
	return &storage.StorageVolume{
		Type:   "bind",
		Source: "apples",
		Target: "pears",
	}, nil
}
