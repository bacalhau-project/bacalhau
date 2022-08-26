package devstack

import (
	"context"
	"fmt"
	"os"
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
	"github.com/stretchr/testify/suite"
)

type DeterministicVerifierSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDeterministicVerifierSuite(t *testing.T) {
	suite.Run(t, new(DeterministicVerifierSuite))
}

// Before all suite
func (suite *DeterministicVerifierSuite) SetupAllSuite() {

}

// Before each test
func (suite *DeterministicVerifierSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *DeterministicVerifierSuite) TearDownTest() {
}

func (suite *DeterministicVerifierSuite) TearDownAllSuite() {

}

type testArgs struct {
	nodeCount      int
	shardCount     int
	badActors      int
	confidence     int
	expectedPassed int
	expectedFailed int
}

// test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *DeterministicVerifierSuite) TestDeterministicVerifier() {
	runTest := func(args testArgs) {
		cm := system.NewCleanupManager()
		ctx := context.Background()
		defer cm.Cleanup()
		getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
			return executor_util.NewNoopStorageProviders(cm, noop_storage.StorageConfig{
				ExternalHooks: noop_storage.StorageConfigExternalHooks{
					Explode: func(ctx context.Context, storageSpec storage.StorageSpec) ([]storage.StorageSpec, error) {
						results := []storage.StorageSpec{}
						for i := 0; i < args.shardCount; i++ {
							results = append(results, storage.StorageSpec{
								Engine: storage.StorageSourceIPFS,
								Cid:    fmt.Sprintf("123%d", i),
								Path:   fmt.Sprintf("/data/file%d.txt", i),
							})
						}
						return results, nil
					},
					// PrepareStorage: func(ctx context.Context, storageSpec storage.StorageSpec) (storage.StorageVolume, error) {
					// 	return storage.StorageVolume{}, nil
					// },
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
							return os.WriteFile(fmt.Sprintf("%s/stdout", resultsDir), []byte(jobStdout), 0644)
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
			args.nodeCount,
			args.badActors,
			getStorageProviders,
			getExecutors,
			getVerifiers,
			getPublishers,
			computenode.NewDefaultComputeNodeConfig(),
			"",
			false,
		)
		require.NoError(suite.T(), err)

		jobSpec := executor.JobSpec{
			Engine:    executor.EngineDocker,
			Verifier:  verifier.VerifierDeterministic,
			Publisher: publisher.PublisherNoop,
			Docker: executor.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					`echo hello`,
				},
			},
			Inputs: []storage.StorageSpec{
				{
					Engine: storage.StorageSourceIPFS,
					Cid:    "123",
				},
			},
			Outputs: []storage.StorageSpec{},
			Sharding: executor.JobShardingConfig{
				GlobPattern: "/data/*.txt",
				BatchSize:   1,
			},
		}

		jobDeal := executor.JobDeal{
			Concurrency: args.nodeCount,
			Confidence:  args.confidence,
		}

		// wait for other nodes to catch up
		time.Sleep(time.Second * 1)
		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)
		submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
		require.NoError(suite.T(), err)

		resolver := apiClient.GetJobStateResolver()

		err = resolver.Wait(
			ctx,
			submittedJob.ID,
			args.nodeCount*args.shardCount,
			job.WaitThrowErrors([]executor.JobStateType{
				executor.JobStateCancelled,
				executor.JobStateError,
			}),
			job.WaitForJobStates(map[executor.JobStateType]int{
				executor.JobStatePublished: args.nodeCount * args.shardCount,
			}),
		)
		require.NoError(suite.T(), err)

		state, err := resolver.GetJobState(ctx, submittedJob.ID)
		require.NoError(suite.T(), err)

		verifiedCount := 0
		failedCount := 0

		for _, state := range state.Nodes {
			for _, shard := range state.Shards {
				require.True(suite.T(), shard.VerificationResult.Complete)
				if shard.VerificationResult.Result {
					verifiedCount++
				} else {
					failedCount++
				}
			}
		}

		require.Equal(suite.T(), args.expectedPassed*args.shardCount, verifiedCount, "verified count should be correct")
		require.Equal(suite.T(), args.expectedFailed*args.shardCount, failedCount, "failed count should be correct")
	}

	// test that we must have more than one node to run the job
	runTest(testArgs{
		nodeCount:      1,
		shardCount:     2,
		badActors:      0,
		confidence:     0,
		expectedPassed: 0,
		expectedFailed: 1,
	})

	// test that if all nodes agree then all are verified
	runTest(testArgs{
		nodeCount:      3,
		shardCount:     2,
		badActors:      0,
		confidence:     0,
		expectedPassed: 3,
		expectedFailed: 0,
	})

	// test that if one node mis-behaves we catch it but the others are verified
	runTest(testArgs{
		nodeCount:      3,
		shardCount:     2,
		badActors:      1,
		confidence:     0,
		expectedPassed: 2,
		expectedFailed: 1,
	})

	// test that is there is a draw between good and bad actors then none are verified
	runTest(testArgs{
		nodeCount:      2,
		shardCount:     2,
		badActors:      1,
		confidence:     0,
		expectedPassed: 0,
		expectedFailed: 2,
	})

	// test that with a larger group the confidence setting gives us a lower threshold
	runTest(testArgs{
		nodeCount:      5,
		shardCount:     2,
		badActors:      2,
		confidence:     4,
		expectedPassed: 0,
		expectedFailed: 5,
	})

}
