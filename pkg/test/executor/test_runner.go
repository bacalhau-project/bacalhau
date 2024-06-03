package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/requester/jobtransform"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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

	fsr, c := setup.SetupBacalhauRepoForTesting(t)
	stack := testutils.Setup(ctx, t, fsr, c, devstack.WithNumberOfHybridNodes(testNodeCount))
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
	execution.AllocateResources(newJob.Task().Name, models.Resources{})

	// Since we are submitting jobs directly to the executor, we need to
	// transform wasm storage specs, which is currently handled by the
	// requester's endpoint
	jobTransformers := []jobtransform.PostTransformer{
		jobtransform.NewWasmStorageSpecConverter(),
	}
	for _, transformer := range jobTransformers {
		_, err = transformer(ctx, newJob)
		require.NoError(t, err)
	}

	resultsDirectory := t.TempDir()
	strgProvider := stack.Nodes[0].ComputeNode.Storages

	runCommandArguments, cleanup, err := compute.PrepareRunArguments(ctx, strgProvider, t.TempDir(), execution, resultsDirectory)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := cleanup(ctx); err != nil {
			t.Fatal(err)
		}
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
