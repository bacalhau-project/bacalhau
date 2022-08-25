package devstack

import (
	"context"
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

// test that the combo driver gives preference to the filecoin unsealed driver
// also that this does not affect normal jobs where the CID resides on the IPFS driver
func (suite *DeterministicVerifierSuite) TestDeterministicVerifier() {
	runTest := func(
		nodeCount int,
		badActors int,
		expectedPassed int,
		expectedFailed int,
	) {
		cm := system.NewCleanupManager()
		ctx := context.Background()
		defer cm.Cleanup()
		getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
			return executor_util.NewNoopStorageProviders(cm)
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
			nodeCount,
			badActors,
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
					"bash", "-c",
					`echo hello`,
				},
			},
			Inputs:  []storage.StorageSpec{},
			Outputs: []storage.StorageSpec{},
		}

		jobDeal := executor.JobDeal{
			Concurrency: nodeCount,
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
			nodeCount,
			job.WaitThrowErrors([]executor.JobStateType{
				executor.JobStateCancelled,
				executor.JobStateError,
			}),
			job.WaitForJobStates(map[executor.JobStateType]int{
				executor.JobStatePublished: nodeCount,
			}),
		)
		require.NoError(suite.T(), err)

		state, err := resolver.GetJobState(ctx, submittedJob.ID)
		require.NoError(suite.T(), err)

		verifiedCount := 0
		failedCount := 0

		for _, state := range state.Nodes {
			shard, ok := state.Shards[0]
			require.True(suite.T(), ok)
			require.True(suite.T(), shard.VerificationResult.Complete)
			if shard.VerificationResult.Result {
				verifiedCount++
			} else {
				failedCount++
			}
		}

		require.Equal(suite.T(), expectedPassed, verifiedCount, "verified count should be correct")
		require.Equal(suite.T(), expectedFailed, failedCount, "failed count should be correct")
	}

	// runTest(1, 0, 1, 0)
	// runTest(3, 0, 3, 0)
	runTest(3, 1, 2, 1)
}
