package ipfs

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestIpfsDockerExecutor(t *testing.T) {

	EXAMPLE_TEXT := `hello world`

	stack, cancelFunction := setupTest(
		t,
		1,
	)

	defer teardownTest(stack, cancelFunction)

	fileCid, err := stack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	assert.NoError(t, err)

	ipfsFuseStorage, err := fuse_docker.NewIpfsFuseDocker(stack.Ctx, stack.Nodes[0].IpfsNode.ApiAddress())
	assert.NoError(t, err)

	dockerExecutor, err := docker.NewDockerExecutor(stack.Ctx, map[string]storage.StorageProvider{
		storage.IPFS_FUSE_DOCKER: ipfsFuseStorage,
	})
	assert.NoError(t, err)

	storageSpec := types.StorageSpec{
		Engine: storage.IPFS_FUSE_DOCKER,
		Cid:    fileCid,
		Path:   "/data/file.txt",
	}

	job := &types.Job{
		Id:    "test-job",
		Owner: "test-owner",
		Spec: &types.JobSpec{
			Engine: executor.EXECUTOR_DOCKER,
			Vm: types.JobSpecVm{
				Image:      "ubuntu",
				Entrypoint: "bash -c 'cat /data/file.txt'",
			},
			Inputs: []types.StorageSpec{
				storageSpec,
			},
			Outputs: []types.StorageSpec{
				{
					Path: "stdout",
				},
			},
		},
		Deal: &types.JobDeal{
			Concurrency:   1,
			AssignedNodes: []string{},
		},
	}

	isInstalled, err := dockerExecutor.IsInstalled()
	assert.NoError(t, err)
	assert.True(t, isInstalled)

	hasStorage, err := dockerExecutor.HasStorage(storageSpec)
	assert.NoError(t, err)
	assert.True(t, hasStorage)

	spew.Dump(job)
}
