//go:build integration || !unit

package compute

import (
	"context"
	"time"

	"github.com/google/uuid"
	libp2p2 "github.com/libp2p/go-libp2p"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	repo2 "github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/system/cleanup"
)

type ComputeSuite struct {
	suite.Suite
	node             *node.Compute
	config           node.ComputeConfig
	cm               *cleanup.CleanupManager
	executor         *noop_executor.NoopExecutor
	publisher        *noop_publisher.NoopPublisher
	stateResolver    resolver.StateResolver
	bidChannel       chan compute.BidResult
	failureChannel   chan compute.ComputeError
	completedChannel chan compute.RunResult
}

func (s *ComputeSuite) SetupSuite() {
	s.config = node.NewComputeConfigWith(node.ComputeConfigParams{
		TotalResourceLimits: model.ResourceUsageData{
			CPU: 2,
		},
	})
}

func (s *ComputeSuite) SetupTest() {
	var err error
	ctx := context.Background()
	s.cm = cleanup.NewCleanupManager()
	s.T().Cleanup(func() { s.cm.Cleanup(ctx) })

	s.executor = noop_executor.NewNoopExecutor()
	s.Require().NoError(err)
	s.publisher = noop_publisher.NewNoopPublisher()
	s.bidChannel = make(chan compute.BidResult)
	s.completedChannel = make(chan compute.RunResult)
	s.failureChannel = make(chan compute.ComputeError)
	s.setupNode()
}

func (s *ComputeSuite) setupNode() {
	repo := repo2.SetupBacalhauRepoForTesting(s.T())
	libp2pPort, err := freeport.GetFreePort()
	s.NoError(err)

	// TODO(forrest) [config] generate a key
	privKey, err := config.GetLibp2pPrivKey()
	s.Require().NoError(err)
	host, err := libp2p.NewHost(libp2pPort, libp2p2.Identity(privKey))
	s.NoError(err)
	s.T().Cleanup(func() { _ = host.Close })

	apiServer, err := publicapi.NewAPIServer(publicapi.APIServerParams{
		Address: "0.0.0.0",
		Port:    0,
		Host:    host,
		Config:  publicapi.DefaultAPIServerConfig,
	})
	s.NoError(err)

	noopstorage := noop_storage.NewNoopStorage()
	s.node, err = node.NewComputeNode(
		context.Background(),
		s.cm,
		host,
		apiServer,
		s.config,
		model.NewNoopProvider[model.StorageSourceType, storage.Storage](noopstorage),
		model.NewNoopProvider[model.Engine, executor.Executor](s.executor),
		model.NewNoopProvider[model.Publisher, publisher.Publisher](s.publisher),
		repo,
	)
	s.NoError(err)
	s.stateResolver = *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})

	s.node.RegisterLocalComputeCallback(compute.CallbackMock{
		OnBidCompleteHandler: func(ctx context.Context, result compute.BidResult) {
			s.bidChannel <- result
		},
		OnRunCompleteHandler: func(ctx context.Context, result compute.RunResult) {
			s.completedChannel <- result
		},
		OnComputeFailureHandler: func(ctx context.Context, err compute.ComputeError) {
			s.failureChannel <- err
		},
	})
	s.T().Cleanup(func() { close(s.bidChannel) })
}

func (s *ComputeSuite) askForBid(ctx context.Context, job model.Job) compute.BidResult {
	_, err := s.node.LocalEndpoint.AskForBid(ctx, compute.AskForBidRequest{
		ExecutionMetadata: compute.ExecutionMetadata{
			JobID:       job.Metadata.ID,
			ExecutionID: uuid.NewString(),
		},
		RoutingMetadata: compute.RoutingMetadata{
			TargetPeerID: s.node.ID,
			SourcePeerID: s.node.ID,
		},
		Job:             job,
		WaitForApproval: true,
	})
	s.NoError(err)

	select {
	case result := <-s.bidChannel:
		return result
	case <-time.After(5 * time.Second):
		s.FailNow("did not receive a bid response")
		return compute.BidResult{}
	}
}

func (s *ComputeSuite) prepareAndAskForBid(ctx context.Context, job model.Job) string {
	result := s.askForBid(ctx, job)
	s.True(result.Accepted)
	return result.ExecutionID
}

func (s *ComputeSuite) prepareAndRun(ctx context.Context, job model.Job) string {
	executionID := s.prepareAndAskForBid(ctx, job)

	// run the job
	_, err := s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCompleted))
	s.NoError(err)
	return executionID
}
