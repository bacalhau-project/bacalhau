package docker

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/test/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

const TEST_STORAGE_DRIVER_NAME = "testdriver"
const TEST_NODE_COUNT = 1

func dockerExecutorStorageTest(
	t *testing.T,
	testCase scenario.TestCase,
	storageDriverFactories []scenario.StorageDriverFactory,
) {

	// the inner test handler that is given the storage driver factory
	// and output mode that we are looping over internally
	runTest := func(
		getStorageDriver scenario.IGetStorageDriver,
	) {

		stack, ctx, cancel := ipfs.SetupTest(t, TEST_NODE_COUNT)
		defer ipfs.TeardownTest(stack, cancel)

		storageDriver, err := getStorageDriver(stack)
		assert.NoError(t, err)

		dockerExecutor, err := docker.NewDockerExecutor(
			ctx, "dockertest", map[string]storage.StorageProvider{
				TEST_STORAGE_DRIVER_NAME: storageDriver,
			})
		assert.NoError(t, err)

		inputStorageList, err := testCase.SetupStorage(
			stack, TEST_STORAGE_DRIVER_NAME, TEST_NODE_COUNT)
		assert.NoError(t, err)

		isInstalled, err := dockerExecutor.IsInstalled()
		assert.NoError(t, err)
		assert.True(t, isInstalled)

		for _, inputStorageSpec := range inputStorageList {
			hasStorage, err := dockerExecutor.HasStorage(inputStorageSpec)
			assert.NoError(t, err)
			assert.True(t, hasStorage)
		}

		job := &types.Job{
			Id:    "test-job",
			Owner: "test-owner",
			Spec: &types.JobSpec{
				Engine:  string(executor.EXECUTOR_DOCKER),
				Vm:      testCase.GetJobSpec(),
				Inputs:  inputStorageList,
				Outputs: testCase.Outputs,
			},
			Deal: &types.JobDeal{
				Concurrency:   TEST_NODE_COUNT,
				AssignedNodes: []string{},
			},
		}

		resultsDirectory, err := dockerExecutor.RunJob(job)
		assert.NoError(t, err)

		if err != nil {
			t.FailNow()
		}

		testCase.ResultsChecker(resultsDirectory)
	}

	for _, storageDriverFactory := range storageDriverFactories {
		log.Debug().Msgf("Running test %s with storage driver %s",
			testCase.Name, storageDriverFactory.Name)
		runTest(storageDriverFactory.DriverFactory)
	}
}

func TestCatFileStdout(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.CatFileToStdout(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestCatFileOutputVolume(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.CatFileToVolume(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestGrepFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.GrepFile(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestSedFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.SedFile(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}

func TestAwkFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.AwkFile(t),
		scenario.STORAGE_DRIVER_FACTORIES,
	)
}
