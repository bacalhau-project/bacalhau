package devstack

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/requesternode"

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
	computeConfig node.ComputeConfig,
	requesterNodeConfig requesternode.RequesterNodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	require.NoError(t, system.InitConfigForTesting(t))

	cm := system.NewCleanupManager()

	options := devstack.DevStackOptions{
		NumberOfNodes:     nodes,
		NumberOfBadActors: badActors,
		LocalNetworkLotus: lotusNode,
	}

	stack, err := devstack.NewStandardDevStack(ctx, cm, options, computeConfig, requesterNodeConfig)
	require.NoError(t, err)

	t.Cleanup(cm.Cleanup)

	// important to give the pubsub network time to connect
	time.Sleep(time.Second)

	return stack, cm
}

func prepareFolderWithFiles(t *testing.T, fileCount int) string { //nolint:unused
	basePath := t.TempDir()
	for i := 0; i < fileCount; i++ {
		err := os.WriteFile(
			fmt.Sprintf("%s/%d.txt", basePath, i),
			[]byte(fmt.Sprintf("hello %d", i)),
			os.ModePerm,
		)
		require.NoError(t, err)
	}
	return basePath
}

type DeterministicVerifierTestArgs struct {
	NodeCount      int
	ShardCount     int
	BadActors      int
	Confidence     int
	ExpectedPassed int
	ExpectedFailed int
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
		return executor_util.NewNoopExecutors(noop_executor.ExecutorConfig{
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
		}), nil
	})

	injector := node.NewStandardNodeDependencyInjector()
	injector.ExecutorsFactory = executorsFactory
	injector.StorageProvidersFactory = storageProvidersFactory

	stack, err := devstack.NewDevStack(
		ctx,
		cm,
		options,
		node.NewComputeConfigWithDefaults(),
		requesternode.NewDefaultRequesterNodeConfig(),
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
