package executor

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
)

func NewDockerIPFSExecutors(ctx context.Context, ipfsMultiAddress string,
	dockerId string) (map[string]Executor, error) {

	ipfsFuseStorage, err := fuse_docker.NewIpfsFuseDocker(ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	ipfsApiCopyStorage, err := api_copy.NewIpfsApiCopy(ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	dockerExecutor, err := docker.NewDockerExecutor(ctx, dockerId,
		map[string]storage.StorageProvider{
			storage.IPFS_FUSE_DOCKER: ipfsFuseStorage,
			storage.IPFS_API_COPY:    ipfsApiCopyStorage,
			// we make the copy driver the "default" storage driver for docker
			// users have to specify the fuse driver explicitly
			storage.IPFS_DEFAULT: ipfsApiCopyStorage,
		})
	if err != nil {
		return nil, err
	}

	return map[string]Executor{
		string(EXECUTOR_DOCKER): dockerExecutor,
	}, nil
}
