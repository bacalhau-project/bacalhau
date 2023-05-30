//go:build integration || !unit

package compute

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
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
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	noop_verifier "github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
)

type ComputeSuite struct {
	suite.Suite
	node          *node.Compute
	config        node.ComputeConfig
	cm            *system.CleanupManager
	executor      *noop_executor.NoopExecutor
	verifier      *noop_verifier.NoopVerifier
	publisher     *noop_publisher.NoopPublisher
	stateResolver resolver.StateResolver
	bidChannel    chan compute.BidResult
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
	s.cm = system.NewCleanupManager()
	s.T().Cleanup(func() { s.cm.Cleanup(ctx) })

	s.executor = noop_executor.NewNoopExecutor()
	s.verifier, err = noop_verifier.NewNoopVerifier(ctx, s.cm)
	s.Require().NoError(err)
	s.publisher = noop_publisher.NewNoopPublisher()
	s.bidChannel = make(chan compute.BidResult)
	s.setupNode()
}

func (s *ComputeSuite) setupNode() {
	libp2pPort, err := freeport.GetFreePort()
	s.NoError(err)

	host, err := libp2p.NewHost(libp2pPort)
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
		"",
		nil,
		model.NewNoopProvider[cid.Cid, storage.Storage](noopstorage),
		model.NewNoopProvider[cid.Cid, executor.Executor](s.executor),
		model.NewNoopProvider[model.Verifier, verifier.Verifier](s.verifier),
		model.NewNoopProvider[model.Publisher, publisher.Publisher](s.publisher),
	)
	s.NoError(err)
	s.stateResolver = *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})

	s.node.RegisterLocalComputeCallback(compute.CallbackMock{
		OnBidCompleteHandler: func(ctx context.Context, result compute.BidResult) {
			s.bidChannel <- result
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
		Job: job,
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
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateWaitingVerification))
	s.NoError(err)
	return executionID
}
