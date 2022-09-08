package docker

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
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
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *ExecutorDockerExecutorSuite) TearDownTest() {

}

func (suite *ExecutorDockerExecutorSuite) TearDownAllSuite() {

}

// const TEST_STORAGE_DRIVER_NAME = "testdriver"
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

		stack := testutils.NewDockerIpfsStack(ctx, t, computenode.NewDefaultComputeNodeConfig())
		defer stack.Node.CleanupManager.Cleanup()

		dockerExecutor := stack.Node.Executors[model.EngineDocker]

		inputStorageList, err := testCase.SetupStorage(ctx,
			model.StorageSourceIPFS, stack.IpfsStack.IPFSClients[:TEST_NODE_COUNT]...)
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

		job := model.Job{
			ID:              "test-job",
			RequesterNodeID: "test-owner",
			ClientID:        "test-client",
			Spec: model.JobSpec{
				Engine:  model.EngineDocker,
				Docker:  testCase.GetJobSpec(),
				Inputs:  inputStorageList,
				Outputs: testCase.Outputs,
			},
			Deal: model.JobDeal{
				Concurrency: TEST_NODE_COUNT,
			},
			CreatedAt: time.Now(),
		}

		shard := model.JobShard{
			Job:   job,
			Index: 0,
		}

		resultsDirectory, err := ioutil.TempDir("", "bacalhau-dockerExecutorStorageTest")
		require.NoError(t, err)

		err = dockerExecutor.RunShard(ctx, shard, resultsDirectory)
		require.NoError(t, err)

		err = testCase.ResultsChecker(resultsDirectory)
		require.NoError(t, err)
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
		scenario.CatFileToStdout(),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestCatFileOutputVolume() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.CatFileToVolume(),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestGrepFile() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.GrepFile(),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestSedFile() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.SedFile(),
		scenario.StorageDriverFactories,
	)
}

func (suite *ExecutorDockerExecutorSuite) TestAwkFile() {
	dockerExecutorStorageTest(
		suite.T(),
		scenario.AwkFile(),
		scenario.StorageDriverFactories,
	)
}
