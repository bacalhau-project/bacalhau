package docker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExecutorDockerExecutorSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestExecutorDockerExecutorSuite(t *testing.T) {
	suite.Run(t, new(ExecutorDockerExecutorSuite))
}

// Before all suite
func (suite *ExecutorDockerExecutorSuite) SetupAllSuite() {

}

// Before each test
func (suite *ExecutorDockerExecutorSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *ExecutorDockerExecutorSuite) TearDownTest() {
}

func (suite *ExecutorDockerExecutorSuite) TearDownAllSuite() {

}

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
		require.NoError(t, err)

		dockerExecutor, err := docker.NewExecutor(
			cm,
			fmt.Sprintf("dockertest-%s", stack.Nodes[0].IpfsNode.ID()),
			map[storage.StorageSourceType]storage.StorageProvider{
				storage.StorageSourceIPFS: storageDriver,
			})
		require.NoError(t, err)

		inputStorageList, err := testCase.SetupStorage(
			stack, storage.StorageSourceIPFS, TEST_NODE_COUNT)
		require.NoError(t, err)

		isInstalled, err := dockerExecutor.IsInstalled(ctx)
		require.NoError(t, err)
		require.True(t, isInstalled)

		for _, inputStorageSpec := range inputStorageList {
			hasStorage, err := dockerExecutor.HasStorageLocally(
				ctx, inputStorageSpec)
			require.NoError(t, err)
			require.True(t, hasStorage)
		}

		job := executor.Job{
			ID:    "test-job",
			Owner: "test-owner",
			Spec: executor.JobSpec{
				Engine:  executor.EngineDocker,
				Docker:  testCase.GetJobSpec(),
				Inputs:  inputStorageList,
				Outputs: testCase.Outputs,
			},
			Deal: executor.JobDeal{
				Concurrency:   TEST_NODE_COUNT,
				AssignedNodes: []string{},
			},
			CreatedAt: time.Now(),
		}

		resultsDirectory, err := dockerExecutor.RunJob(ctx, job)
		require.NoError(t, err)

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

func (suite *ExecutorDockerExecutorSuite) TestCatFileStdout() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.CatFileToStdout(suite.T()),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestCatFileOutputVolume() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.CatFileToVolume(suite.T()),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestGrepFile() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.GrepFile(suite.T()),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestSedFile() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.SedFile(suite.T()),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestAwkFile() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.AwkFile(suite.T()),
		scenario.StorageDriverFactories,
	)
}
