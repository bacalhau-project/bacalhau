package docker

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

func TestIpfsDockerExecutor(t *testing.T) {

	EXAMPLE_TEXT := `hello world`
	MOUNT_PATH := "/data/file.txt"

	fuseStorageDriverFactory := func(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error) {
		return fuse_docker.NewIpfsFuseDocker(stack.Ctx, stack.Nodes[0].IpfsNode.ApiAddress())
	}

	apiCopyStorageDriverFactory := func(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error) {
		return api_copy.NewIpfsApiCopy(stack.Ctx, stack.Nodes[0].IpfsNode.ApiAddress())
	}

	getDataTest := func(stack *devstack.DevStack_IPFS) *IDataBasedTest {
		return dataBasedTestSingleFile(
			t,
			stack,
			EXAMPLE_TEXT,
			MOUNT_PATH,
			"stdout",
		)
	}

	jobSpec := types.JobSpecVm{
		Image: "ubuntu",
		Entrypoint: []string{
			"cat",
			MOUNT_PATH,
		},
	}

	dockerExecutorStorageTest(
		t,
		fuseStorageDriverFactory,
		getDataTest,
		jobSpec,
	)

	dockerExecutorStorageTest(
		t,
		apiCopyStorageDriverFactory,
		getDataTest,
		jobSpec,
	)
}
