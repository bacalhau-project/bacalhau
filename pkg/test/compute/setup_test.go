package compute

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/compute/store/resolver"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	noop_publisher "github.com/filecoin-project/bacalhau/pkg/publisher/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	noop_verifier "github.com/filecoin-project/bacalhau/pkg/verifier/noop"
	"github.com/stretchr/testify/suite"
)

type ComputeSuite struct {
	suite.Suite
	node          *node.Compute
	config        node.ComputeConfig
	jobStore      localdb.LocalDB
	executor      *noop_executor.NoopExecutor
	verifier      *noop_verifier.NoopVerifier
	publisher     *noop_publisher.NoopPublisher
	stateResolver resolver.StateResolver
}

func (s *ComputeSuite) SetupTest() {
	ctx := context.Background()
	cm := system.NewCleanupManager()
	jobStore, err := inmemory.NewInMemoryDatastore()
	s.NoError(err)

	s.jobStore = jobStore
	s.config = node.NewComputeConfigWith(node.ComputeConfigParams{
		TotalResourceLimits: model.ResourceUsageData{
			CPU: 2,
		},
		OverCommitResourcesFactor: 1.5,
	})
	s.executor = noop_executor.NewNoopExecutor()
	s.verifier, err = noop_verifier.NewNoopVerifier(ctx, cm, localdb.GetStateResolver(s.jobStore))
	s.publisher = noop_publisher.NewNoopPublisher()
	s.setupNode()
}

func (s *ComputeSuite) setupNode() {
	s.node = node.NewComputeNode(
		context.Background(),
		s.T().Name(),
		s.config,
		s.jobStore,
		noop_executor.NewNoopExecutorProvider(s.executor),
		noop_verifier.NewNoopVerifierProvider(s.verifier),
		noop_publisher.NewNoopPublisherProvider(s.publisher),
		eventhandler.NewDefaultTracer(),
	)
	s.stateResolver = *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})
}

func TestComputeSuite(t *testing.T) {
	suite.Run(t, new(ComputeSuite))
}

func (s *ComputeSuite) prepareAndAskForBid(ctx context.Context, job model.Job) string {
	response, err := s.node.Frontend.AskForBid(ctx, frontend.AskForBidRequest{
		Job:          job,
		ShardIndexes: []int{0},
	})
	s.NoError(err)

	// check the response
	s.Equal(1, len(response.ShardResponse))
	s.True(response.ShardResponse[0].Accepted)

	return response.ShardResponse[0].ExecutionID
}

func (s *ComputeSuite) prepareAndRun(ctx context.Context, job model.Job) string {
	executionID := s.prepareAndAskForBid(ctx, job)

	// run the job
	_, err := s.node.Frontend.BidAccepted(ctx, frontend.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateWaitingVerification))
	s.NoError(err)
	return executionID
}
