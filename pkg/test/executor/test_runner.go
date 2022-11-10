package executor

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
)

const testNodeCount = 1

func RunTestCase(
	t *testing.T,
	testCase scenario.Scenario,
) {
	ctx := context.Background()
	spec := testCase.Spec

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

		storageList, stErr := getStorage(ctx,
			model.StorageSourceIPFS, stack.IpfsStack.IPFSClients[:testNodeCount]...)
		require.NoError(t, stErr)

		for _, storageSpec := range storageList {
			hasStorage, stErr := executor.HasStorageLocally(
				ctx, storageSpec)
			require.NoError(t, stErr)
			require.True(t, hasStorage)
		}

		return storageList
	}

	spec.Inputs = prepareStorage(testCase.Inputs)
	spec.Contexts = prepareStorage(testCase.Contexts)
	spec.Outputs = testCase.Outputs

	job := &model.Job{
		ID:              "test-job",
		RequesterNodeID: "test-owner",
		ClientID:        "test-client",
		Spec:            spec,
		Deal: model.Deal{
			Concurrency: testNodeCount,
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
