package devstack

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

func SetupTest(
	ctx context.Context,
	t *testing.T,
	nodes int, badActors int,
	lotusNode bool,
	//nolint:gocritic
	config computenode.ComputeNodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	require.NoError(t, system.InitConfigForTesting())

	cm := system.NewCleanupManager()

	options := devstack.DevStackOptions{
		NumberOfNodes:     nodes,
		NumberOfBadActors: badActors,
		LocalNetworkLotus: lotusNode,
	}

	stack, err := devstack.NewStandardDevStack(ctx, cm, options, config)
	require.NoError(t, err)

	t.Cleanup(cm.Cleanup)

	// important to give the pubsub network time to connect
	time.Sleep(time.Second)

	return stack, cm
}

type DeterministicVerifierTestArgs struct {
	NodeCount      int
	ShardCount     int
	BadActors      int
	Confidence     int
	ExpectedPassed int
	ExpectedFailed int
}

func RunDeterministicVerifierTests(
	ctx context.Context,
	t *testing.T,
	submitJob func(
		apiClient *publicapi.APIClient,
		args DeterministicVerifierTestArgs,
	) (string, error),
) {
	// test that we must have more than one node to run the job
	t.Run("more-than-one-node-to-run-the-job", func(t *testing.T) {
		RunDeterministicVerifierTest(ctx, t, submitJob, DeterministicVerifierTestArgs{
			NodeCount:      1,
			ShardCount:     2,
			BadActors:      0,
			Confidence:     0,
			ExpectedPassed: 0,
			ExpectedFailed: 1,
		})
	})

	// test that if all nodes agree then all are verified
	t.Run("all-nodes-agree-then-all-are-verified", func(t *testing.T) {
		RunDeterministicVerifierTest(ctx, t, submitJob, DeterministicVerifierTestArgs{
			NodeCount:      3,
			ShardCount:     2,
			BadActors:      0,
			Confidence:     0,
			ExpectedPassed: 3,
			ExpectedFailed: 0,
		})
	})

	// test that if one node mis-behaves we catch it but the others are verified
	t.Run("one-node-misbehaves-but-others-are-verified", func(t *testing.T) {
		RunDeterministicVerifierTest(ctx, t, submitJob, DeterministicVerifierTestArgs{
			NodeCount:      3,
			ShardCount:     2,
			BadActors:      1,
			Confidence:     0,
			ExpectedPassed: 2,
			ExpectedFailed: 1,
		})
	})

	// test that is there is a draw between good and bad actors then none are verified
	t.Run("draw-between-good-and-bad-actors-then-none-are-verified", func(t *testing.T) {
		RunDeterministicVerifierTest(ctx, t, submitJob, DeterministicVerifierTestArgs{
			NodeCount:      2,
			ShardCount:     2,
			BadActors:      1,
			Confidence:     0,
			ExpectedPassed: 0,
			ExpectedFailed: 2,
		})
	})

	// test that with a larger group the confidence setting gives us a lower threshold
	t.Run("larger-group-with-confidence-gives-lower-threshold", func(t *testing.T) {
		RunDeterministicVerifierTest(ctx, t, submitJob, DeterministicVerifierTestArgs{
			NodeCount:      5,
			ShardCount:     2,
			BadActors:      2,
			Confidence:     4,
			ExpectedPassed: 0,
			ExpectedFailed: 5,
		})
	})
}

func RunDeterministicVerifierTest( //nolint:funlen
	ctx context.Context,
	t *testing.T,
	submitJob func(
		apiClient *publicapi.APIClient,
		args DeterministicVerifierTestArgs,
	) (string, error),
	args DeterministicVerifierTestArgs,
) {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	options := devstack.DevStackOptions{
		NumberOfNodes:     args.NodeCount,
		NumberOfBadActors: args.BadActors,
	}

	storageProvidersFactory := devstack.NewNoopStorageProvidersFactoryWithConfig(noop_storage.StorageConfig{
		ExternalHooks: noop_storage.StorageConfigExternalHooks{
			Explode: func(ctx context.Context, storageSpec model.StorageSpec) ([]model.StorageSpec, error) {
				var results []model.StorageSpec
				for i := 0; i < args.ShardCount; i++ {
					results = append(results, model.StorageSpec{
						StorageSource: model.StorageSourceIPFS,
						CID:           fmt.Sprintf("123%d", i),
						Path:          fmt.Sprintf("/data/file%d.txt", i),
					})
				}
				return results, nil
			},
		},
	})

	executorsFactory := node.ExecutorsFactoryFunc(func(
		ctx context.Context, nodeConfig node.NodeConfig) (executor.ExecutorProvider, error) {
		return executor_util.NewNoopExecutors(ctx, cm, noop_executor.ExecutorConfig{
			IsBadActor: nodeConfig.IsBadActor,
			ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
				JobHandler: func(ctx context.Context, shard model.JobShard, resultsDir string) (*model.RunCommandResult, error) {
					runOutput := &model.RunCommandResult{}
					runOutput.STDOUT = fmt.Sprintf("hello world %d", shard.Index)
					if nodeConfig.IsBadActor {
						runOutput.STDOUT = fmt.Sprintf("i am bad and deserve to fail %d", shard.Index)
					}
					err := os.WriteFile(fmt.Sprintf("%s/stdout", resultsDir), []byte(runOutput.STDOUT), 0600) //nolint:gomnd
					if err != nil {
						runOutput.ErrorMsg = err.Error()
					}

					// Adding explicit error for consistency in function signatures
					return runOutput, err
				},
			},
		})
	})

	injector := node.NewStandardNodeDependencyInjector()
	injector.ExecutorsFactory = executorsFactory
	injector.StorageProvidersFactory = storageProvidersFactory

	stack, err := devstack.NewDevStack(
		ctx,
		cm,
		options,
		computenode.NewDefaultComputeNodeConfig(),
		injector,
	)
	require.NoError(t, err)

	// wait for other nodes to catch up
	time.Sleep(time.Second * 1)
	apiURI := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiURI)

	jobID, err := submitJob(apiClient, args)
	require.NoError(t, err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		jobID,
		args.NodeCount*args.ShardCount,
		job.WaitForTerminalStates(args.NodeCount*args.ShardCount),
	)
	require.NoError(t, err)

	state, err := resolver.GetJobState(ctx, jobID)
	require.NoError(t, err)

	verifiedCount := 0
	failedCount := 0

	for _, state := range state.Nodes {
		for _, shard := range state.Shards { //nolint:gocritic
			require.True(t, shard.VerificationResult.Complete)
			if shard.VerificationResult.Result {
				verifiedCount++
			} else {
				failedCount++
			}
		}
	}

	require.Equal(t, args.ExpectedPassed*args.ShardCount, verifiedCount, "verified count should be correct")
	require.Equal(t, args.ExpectedFailed*args.ShardCount, failedCount, "failed count should be correct")
}
