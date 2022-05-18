package executor

import (
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func NewDockerIPFSExecutors(
	cancelContext *system.CancelContext,
	ipfsMultiAddress string,
	dockerId string,
) (map[string]Executor, error) {
	executors := map[string]Executor{}
	ipfsFuseStorage, err := fuse_docker.NewIpfsFuseDocker(cancelContext, ipfsMultiAddress)
	if err != nil {
		return executors, err
	}
	ipfsApiCopyStorage, err := api_copy.NewIpfsApiCopy(cancelContext, ipfsMultiAddress)
	if err != nil {
		return executors, err
	}
	dockerExecutor, err := docker.NewDockerExecutor(cancelContext, dockerId, map[string]storage.StorageProvider{
		storage.IPFS_FUSE_DOCKER: ipfsFuseStorage,
		storage.IPFS_API_COPY:    ipfsApiCopyStorage,
	})
	if err != nil {
		return executors, err
	}
	executors[EXECUTOR_DOCKER] = dockerExecutor
	return executors, nil
}
