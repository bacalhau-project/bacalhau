//go:build !(unit && (windows || darwin))

package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExecutorTestSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}

// Before each test
func (suite *ExecutorTestSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

const TEST_NODE_COUNT = 1

func runTestCase(
	t *testing.T,
	testCase scenario.TestCase,
	getStorageDriver scenario.IGetStorageDriver,
) {
	ctx := context.Background()
	spec := testCase.GetJobSpec()

	stack := testutils.NewDevStack(ctx, t, computenode.NewDefaultComputeNodeConfig())
	defer stack.Node.CleanupManager.Cleanup()

	executor, err := stack.Node.Executors.GetExecutor(ctx, spec.Engine)
	require.NoError(t, err)

	isInstalled, err := executor.IsInstalled(ctx)
	require.NoError(t, err)
	require.True(t, isInstalled)

	prepareStorage := func(getStorage scenario.ISetupStorage) []model.StorageSpec {
		if getStorage == nil {
			return []model.StorageSpec{}
		}

		storageList, err := getStorage(ctx,
			model.StorageSourceIPFS, stack.IpfsStack.IPFSClients[:TEST_NODE_COUNT]...)
		require.NoError(t, err)

		for _, storageSpec := range storageList {
			hasStorage, err := executor.HasStorageLocally(
				ctx, storageSpec)
			require.NoError(t, err)
			require.True(t, hasStorage)
		}

		return storageList
	}

	spec.Inputs = prepareStorage(testCase.SetupStorage)
	spec.Contexts = prepareStorage(testCase.SetupContext)
	spec.Outputs = testCase.Outputs

	job := &model.Job{
		ID:              "test-job",
		RequesterNodeID: "test-owner",
		ClientID:        "test-client",
		Spec:            spec,
		Deal: model.Deal{
			Concurrency: TEST_NODE_COUNT,
		},
		CreatedAt: time.Now(),
	}

	shard := model.JobShard{
		Job:   job,
		Index: 0,
	}

	resultsDirectory := t.TempDir()

	runnerOutput, err := executor.RunShard(ctx, shard, resultsDirectory)
	require.NoError(t, err)
	require.Empty(t, runnerOutput.ErrorMsg)

	err = testCase.ResultsChecker(resultsDirectory)
	require.NoError(t, err)
}

func (suite *ExecutorTestSuite) TestScenarios() {
	for _, testCase := range scenario.GetAllScenarios() {
		for _, storageDriverFactory := range scenario.StorageDriverFactories {
			suite.Run(
				strings.Join([]string{testCase.Name, storageDriverFactory.Name}, "-"),
				func() { runTestCase(suite.T(), testCase, storageDriverFactory.DriverFactory) },
			)
		}
	}
}
