package devstack

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/require"
)

var StorageNames = []storage.StorageSourceType{
	storage.StorageSourceIPFS,
}

func SetupTest(
	t *testing.T,
	nodes int, badActors int,
	//nolint:gocritic
	config computenode.ComputeNodeConfig,
) (*devstack.DevStack, *system.CleanupManager) {
	system.InitConfigForTesting(t)

	cm := system.NewCleanupManager()
	getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
		return executor_util.NewStandardStorageProviders(cm, executor_util.StandardStorageProviderOptions{
			IPFSMultiaddress: ipfsMultiAddress,
		})
	}
	getExecutors := func(
		ipfsMultiAddress string,
		nodeIndex int,
		isBadActor bool,
		ctrl *controller.Controller,
	) (
		map[executor.EngineType]executor.Executor,
		error,
	) {
		ipfsParts := strings.Split(ipfsMultiAddress, "/")
		ipfsSuffix := ipfsParts[len(ipfsParts)-1]
		return executor_util.NewStandardExecutors(
			cm,
			executor_util.StandardExecutorOptions{
				DockerID:   fmt.Sprintf("devstacknode%d-%s", nodeIndex, ipfsSuffix),
				IsBadActor: isBadActor,
				Storage: executor_util.StandardStorageProviderOptions{
					IPFSMultiaddress: ipfsMultiAddress,
				},
			},
		)
	}
	getVerifiers := func(
		transport *libp2p.LibP2PTransport,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[verifier.VerifierType]verifier.Verifier,
		error,
	) {
		return verifier_util.NewStandardVerifiers(
			cm,
			ctrl.GetStateResolver(),
			transport.Encrypt,
			transport.Decrypt,
		)
	}
	getPublishers := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[publisher.PublisherType]publisher.Publisher,
		error,
	) {
		return publisher_util.NewIPFSPublishers(cm, ctrl.GetStateResolver(), ipfsMultiAddress)
	}
	stack, err := devstack.NewDevStack(
		cm,
		nodes,
		badActors,
		getStorageProviders,
		getExecutors,
		getVerifiers,
		getPublishers,
		config,
		"",
		false,
	)
	require.NoError(t, err)

	// important to give the pubsub network time to connect
	time.Sleep(time.Second)

	return stack, cm
}

func TeardownTest(stack *devstack.DevStack, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
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
	t *testing.T,
	submitJob func(
		apiClient *publicapi.APIClient,
		args DeterministicVerifierTestArgs,
	) (string, error),
) {
	// test that we must have more than one node to run the job
	RunDeterministicVerifierTest(t, submitJob, DeterministicVerifierTestArgs{
		NodeCount:      1,
		ShardCount:     2,
		BadActors:      0,
		Confidence:     0,
		ExpectedPassed: 0,
		ExpectedFailed: 1,
	})

	// test that if all nodes agree then all are verified
	RunDeterministicVerifierTest(t, submitJob, DeterministicVerifierTestArgs{
		NodeCount:      3,
		ShardCount:     2,
		BadActors:      0,
		Confidence:     0,
		ExpectedPassed: 3,
		ExpectedFailed: 0,
	})

	// test that if one node mis-behaves we catch it but the others are verified
	RunDeterministicVerifierTest(t, submitJob, DeterministicVerifierTestArgs{
		NodeCount:      3,
		ShardCount:     2,
		BadActors:      1,
		Confidence:     0,
		ExpectedPassed: 2,
		ExpectedFailed: 1,
	})

	// test that is there is a draw between good and bad actors then none are verified
	RunDeterministicVerifierTest(t, submitJob, DeterministicVerifierTestArgs{
		NodeCount:      2,
		ShardCount:     2,
		BadActors:      1,
		Confidence:     0,
		ExpectedPassed: 0,
		ExpectedFailed: 2,
	})

	// test that with a larger group the confidence setting gives us a lower threshold
	RunDeterministicVerifierTest(t, submitJob, DeterministicVerifierTestArgs{
		NodeCount:      5,
		ShardCount:     2,
		BadActors:      2,
		Confidence:     4,
		ExpectedPassed: 0,
		ExpectedFailed: 5,
	})
}

func RunDeterministicVerifierTest( //nolint:funlen
	t *testing.T,
	submitJob func(
		apiClient *publicapi.APIClient,
		args DeterministicVerifierTestArgs,
	) (string, error),
	args DeterministicVerifierTestArgs,
) {
	cm := system.NewCleanupManager()
	ctx := context.Background()
	defer cm.Cleanup()
	getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
		return executor_util.NewNoopStorageProviders(cm, noop_storage.StorageConfig{
			ExternalHooks: noop_storage.StorageConfigExternalHooks{
				Explode: func(ctx context.Context, storageSpec storage.StorageSpec) ([]storage.StorageSpec, error) {
					results := []storage.StorageSpec{}
					for i := 0; i < args.ShardCount; i++ {
						results = append(results, storage.StorageSpec{
							Engine: storage.StorageSourceIPFS,
							Cid:    fmt.Sprintf("123%d", i),
							Path:   fmt.Sprintf("/data/file%d.txt", i),
						})
					}
					return results, nil
				},
			},
		})
	}
	getExecutors := func(
		ipfsMultiAddress string,
		nodeIndex int,
		isBadActor bool,
		ctrl *controller.Controller,
	) (map[executor.EngineType]executor.Executor, error) {
		return executor_util.NewNoopExecutors(
			cm,
			noop_executor.ExecutorConfig{
				IsBadActor: isBadActor,
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					JobHandler: func(ctx context.Context, job executor.Job, shardIndex int, resultsDir string) error {
						jobStdout := fmt.Sprintf("hello world %d", shardIndex)
						if isBadActor {
							jobStdout = fmt.Sprintf("i am bad and deserve to fail %d", shardIndex)
						}
						return os.WriteFile(fmt.Sprintf("%s/stdout", resultsDir), []byte(jobStdout), 0600) //nolint:gomnd
					},
				},
			},
		)
	}
	getVerifiers := func(
		transport *libp2p.LibP2PTransport,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[verifier.VerifierType]verifier.Verifier,
		error,
	) {
		return verifier_util.NewStandardVerifiers(
			cm,
			ctrl.GetStateResolver(),
			transport.Encrypt,
			transport.Decrypt,
		)
	}
	getPublishers := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[publisher.PublisherType]publisher.Publisher,
		error,
	) {
		return publisher_util.NewNoopPublishers(cm, ctrl.GetStateResolver())
	}
	stack, err := devstack.NewDevStack(
		cm,
		args.NodeCount,
		args.BadActors,
		getStorageProviders,
		getExecutors,
		getVerifiers,
		getPublishers,
		computenode.NewDefaultComputeNodeConfig(),
		"",
		false,
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
		job.WaitThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		job.WaitForJobStates(map[executor.JobStateType]int{
			executor.JobStatePublished: args.NodeCount * args.ShardCount,
		}),
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
