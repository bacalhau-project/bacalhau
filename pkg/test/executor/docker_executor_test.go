package docker

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/test/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
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
	runTest := func(getStorageDriver scenario.IGetStorageDriver) {
		ctx := context.Background()
		stack, cm := ipfs.SetupTest(t, TEST_NODE_COUNT)
		defer ipfs.TeardownTest(stack, cm)

		storageDriver, err := getStorageDriver(stack)
		assert.NoError(t, err)

		dockerExecutor, err := docker.NewExecutor(
			cm, "dockertest", map[string]storage.StorageProvider{
				TEST_STORAGE_DRIVER_NAME: storageDriver,
			})
		assert.NoError(t, err)

		inputStorageList, err := testCase.SetupStorage(
			stack, TEST_STORAGE_DRIVER_NAME, TEST_NODE_COUNT)
		assert.NoError(t, err)

		isInstalled, err := dockerExecutor.IsInstalled(ctx)
		assert.NoError(t, err)
		assert.True(t, isInstalled)

		for _, inputStorageSpec := range inputStorageList {
			hasStorage, err := dockerExecutor.HasStorage(
				ctx, inputStorageSpec)
			assert.NoError(t, err)
			assert.True(t, hasStorage)
		}

		job := &executor.Job{
			ID:    "test-job",
			Owner: "test-owner",
			Spec: &executor.JobSpec{
				Engine:  executor.EngineDocker,
				Docker:  testCase.GetJobSpec(),
				Inputs:  inputStorageList,
				Outputs: testCase.Outputs,
			},
			Deal: &executor.JobDeal{
				Concurrency:   TEST_NODE_COUNT,
				AssignedNodes: []string{},
			},
			CreatedAt: time.Now(),
		}

		resultsDirectory, err := dockerExecutor.RunJob(ctx, job)
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
		scenario.StorageDriverFactories,
	)
}

func TestCatFileOutputVolume(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.CatFileToVolume(t),
		scenario.StorageDriverFactories,
	)
}

func TestGrepFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.GrepFile(t),
		scenario.StorageDriverFactories,
	)
}

func TestSedFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.SedFile(t),
		scenario.StorageDriverFactories,
	)
}

func TestAwkFile(t *testing.T) {
	dockerExecutorStorageTest(
		t,
		scenario.AwkFile(t),
		scenario.StorageDriverFactories,
	)
}
