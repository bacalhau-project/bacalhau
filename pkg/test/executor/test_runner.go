package executor

import (
	"context"
	"testing"
	"time"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
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

	stack, _ := testutils.SetupTest(ctx, t, testNodeCount, 0, false,
		node.NewComputeConfigWithDefaults(),
		node.NewRequesterConfigWithDefaults(),
	)
	executor, err := stack.Nodes[0].ComputeNode.Executors.Get(ctx, spec.Engine)
	require.NoError(t, err)

	isInstalled, err := executor.IsInstalled(ctx)
	require.NoError(t, err)
	require.True(t, isInstalled)

	prepareStorage := func(getStorage scenario.SetupStorage) []model.StorageSpec {
		if getStorage == nil {
			return []model.StorageSpec{}
		}

		storageList, stErr := getStorage(ctx,
			model.StorageSourceIPFS, stack.IPFSClients()[:testNodeCount]...)
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
	spec.Deal = model.Deal{Concurrency: testNodeCount}

	job := &model.Job{
		Metadata: model.Metadata{
			ID:        "test-job",
			ClientID:  "test-client",
			CreatedAt: time.Now(),
			Requester: model.JobRequester{
				RequesterNodeID: "test-owner",
			},
		},
		Spec: spec,
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
