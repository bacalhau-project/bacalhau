package docker

import (
	"fmt"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/test/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

const TEST_STORAGE_DRIVER_NAME = "testdriver"

type IDataBasedTest struct {
	setupStorage func() ([]types.StorageSpec, error)
	checkResults func(resultsDir string)
}

// a storage setup function that writes the given string to a single file
func dataBasedTestSingleFile(
	t *testing.T,
	stack *devstack.DevStack_IPFS,
	fileContents string,
	mountPath string,
	resultsPath string,
) *IDataBasedTest {

	setupStorage := func() ([]types.StorageSpec, error) {
		fileCid, err := stack.AddTextToNodes(1, []byte(fileContents))
		assert.NoError(t, err)
		inputStorageSpecs := []types.StorageSpec{
			{
				Engine: TEST_STORAGE_DRIVER_NAME,
				Cid:    fileCid,
				Path:   mountPath,
			},
		}
		return inputStorageSpecs, nil
	}

	checkResults := func(resultsDirectory string) {
		stdout, err := os.ReadFile(fmt.Sprintf("%s/%s", resultsDirectory, resultsPath))
		assert.NoError(t, err)
		assert.Equal(t, string(stdout), fileContents)
	}

	return &IDataBasedTest{
		setupStorage: setupStorage,
		checkResults: checkResults,
	}
}

func dockerExecutorStorageTest(
	t *testing.T,
	getStorageDriver func(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error),
	getDataTest func(stack *devstack.DevStack_IPFS) *IDataBasedTest,
	jobSpec types.JobSpecVm,
) {

	stack, cancelFunction := ipfs.SetupTest(
		t,
		1,
	)

	defer ipfs.TeardownTest(stack, cancelFunction)

	storageDriver, err := getStorageDriver(stack)
	assert.NoError(t, err)

	dockerExecutor, err := docker.NewDockerExecutor(stack.Ctx, "dockertest", map[string]storage.StorageProvider{
		TEST_STORAGE_DRIVER_NAME: storageDriver,
	})
	assert.NoError(t, err)

	dataBasedTest := getDataTest(stack)
	assert.NoError(t, err)

	inputStorageList, err := dataBasedTest.setupStorage()
	assert.NoError(t, err)

	job := &types.Job{
		Id:    "test-job",
		Owner: "test-owner",
		Spec: &types.JobSpec{
			Engine: executor.EXECUTOR_DOCKER,
			Vm:     jobSpec,
			Inputs: inputStorageList,
		},
		Deal: &types.JobDeal{
			Concurrency:   1,
			AssignedNodes: []string{},
		},
	}

	isInstalled, err := dockerExecutor.IsInstalled()
	assert.NoError(t, err)
	assert.True(t, isInstalled)

	for _, inputStorageSpec := range inputStorageList {
		hasStorage, err := dockerExecutor.HasStorage(inputStorageSpec)
		assert.NoError(t, err)
		assert.True(t, hasStorage)
	}

	resultsDirectory, err := dockerExecutor.RunJob(job)
	assert.NoError(t, err)

	dataBasedTest.checkResults(resultsDirectory)
}
