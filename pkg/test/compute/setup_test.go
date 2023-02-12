package compute

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/compute/store/resolver"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	noop_publisher "github.com/filecoin-project/bacalhau/pkg/publisher/noop"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	noop_storage "github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	noop_verifier "github.com/filecoin-project/bacalhau/pkg/verifier/noop"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/suite"
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
}

func (s *ComputeSuite) SetupTest() {
	var err error
	ctx := context.Background()
	s.cm = system.NewCleanupManager()
	s.config = node.NewComputeConfigWith(node.ComputeConfigParams{
		TotalResourceLimits: model.ResourceUsageData{
			CPU: 2,
		},
	})
	s.executor = noop_executor.NewNoopExecutor()
	s.verifier, err = noop_verifier.NewNoopVerifier(ctx, s.cm)
	s.Require().NoError(err)
	s.publisher = noop_publisher.NewNoopPublisher()
	s.setupNode()
}

func (s *ComputeSuite) setupNode() {
	libp2pPort, err := freeport.GetFreePort()
	s.NoError(err)

	host, err := libp2p.NewHost(libp2pPort)
	s.NoError(err)

	apiPort, err := freeport.GetFreePort()
	s.NoError(err)

	apiServer, err := publicapi.NewAPIServer(publicapi.APIServerParams{
		Address: "0.0.0.0",
		Port:    apiPort,
		Host:    host,
		Config:  publicapi.DefaultAPIServerConfig,
	})
	s.NoError(err)

	noopstorage, err := noop_storage.NewNoopStorage(nil, nil, noop_storage.StorageConfig{})
	s.Require().NoError(err)

	s.node, err = node.NewComputeNode(
		context.Background(),
		s.cm,
		host,
		apiServer,
		s.config,
		"",
		nil,
		model.NewNoopProvider[model.StorageSourceType, storage.Storage](noopstorage),
		model.NewNoopProvider[model.Engine, executor.Executor](s.executor),
		model.NewNoopProvider[model.Verifier, verifier.Verifier](s.verifier),
		model.NewNoopProvider[model.Publisher, publisher.Publisher](s.publisher),
	)
	s.NoError(err)
	s.stateResolver = *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})
}

func TestComputeSuite(t *testing.T) {
	suite.Run(t, new(ComputeSuite))
}

func (s *ComputeSuite) prepareAndAskForBid(ctx context.Context, job model.Job) string {
	response, err := s.node.LocalEndpoint.AskForBid(ctx, compute.AskForBidRequest{
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
	_, err := s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateWaitingVerification))
	s.NoError(err)
	return executionID
}
