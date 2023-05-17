package executor

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	exec "github.com/bacalhau-project/bacalhau/pkg/executor"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	noop_verifier "github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
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

	storageProvider := stack.Nodes[0].ComputeNode.Storage

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
	spec.Outputs = testCase.Outputs
	spec.Deal = model.Deal{Concurrency: testNodeCount}

	job := model.Job{
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

	execution := store.Execution{
		ID:  "test-execution",
		Job: job,
	}

	env, err := exec.NewEnvironment(execution, storageProvider)
	require.NoError(t, err)

	cm := system.NewCleanupManager()
	t.Cleanup(func() { cm.Cleanup(context.Background()) })

	resultsDirectory := t.TempDir()

	verifierMock, err := noop_verifier.NewNoopVerifierWithConfig(context.Background(), cm, noop_verifier.VerifierConfig{
		ExternalHooks: noop_verifier.VerifierExternalHooks{
			GetResultPath: func(ctx context.Context, executionID string, job model.Job) (string, error) {
				_ = os.MkdirAll(resultsDirectory, util.OS_ALL_RWX)
				return resultsDirectory, nil
			},
		},
	})
	require.NoError(t, err)

	err = env.Build(ctx, verifierMock)
	require.NoError(t, err)
	defer env.Destroy(ctx) //nolint:errcheck

	runnerOutput, err := executor.Run(ctx, env)
	require.NoError(t, err)
	require.Empty(t, runnerOutput.ErrorMsg)

	err = testCase.ResultsChecker(resultsDirectory)
	require.NoError(t, err)
}
