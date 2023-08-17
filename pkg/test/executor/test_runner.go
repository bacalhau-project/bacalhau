package executor

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

const testNodeCount = 1

func RunTestCase(
	t *testing.T,
	testCase scenario.Scenario,
) {
	ctx := context.Background()
	spec := testCase.Spec

	stack := testutils.Setup(ctx, t, devstack.WithNumberOfHybridNodes(testNodeCount))
	executor, err := stack.Nodes[0].ComputeNode.Executors.Get(ctx, spec.EngineSpec.Engine().String())
	require.NoError(t, err)

	isInstalled, err := executor.IsInstalled(ctx)
	require.NoError(t, err)
	require.True(t, isInstalled)

	prepareStorage := func(getStorage scenario.SetupStorage) []model.StorageSpec {
		if getStorage == nil {
			return []model.StorageSpec{}
		}

		storageList, stErr := getStorage(ctx, model.StorageSourceIPFS, stack.IPFSClients()[:testNodeCount]...)
		require.NoError(t, stErr)

		for _, storageSpec := range storageList {
			inputSource, err := legacy.FromLegacyStorageSpecToInputSource(storageSpec)
			require.NoError(t, err)

			strger, err := stack.Nodes[0].ComputeNode.Storages.Get(ctx, storageSpec.StorageSource.String())
			require.NoError(t, err)
			hasStorage, stErr := strger.HasStorageLocally(ctx, *inputSource)
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

	newJob, err := legacy.FromLegacyJob(&job)
	require.NoError(t, err)
	execution := mock.ExecutionForJob(newJob)
	execution.AllocateResources(newJob.Task().Name, models.Resources{CPU: 1, Memory: 1})

	resultsDirectory := t.TempDir()
	strgProvider := stack.Nodes[0].ComputeNode.Storages
	runCommandCleanup := system.NewCleanupManager()

	runCommandArguments, err := compute.PrepareRunArguments(ctx, strgProvider, execution, resultsDirectory, runCommandCleanup)
	require.NoError(t, err)
	t.Cleanup(func() {
		runCommandCleanup.Cleanup(ctx)
	})

	_, err = executor.Run(ctx, runCommandArguments)
	if testCase.SubmitChecker != nil {
		err = testCase.SubmitChecker(&job, err)
		require.NoError(t, err)
	}

	if testCase.ResultsChecker != nil {
		err = testCase.ResultsChecker(resultsDirectory)
		require.NoError(t, err)
	}
}
