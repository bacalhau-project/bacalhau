//go:build integration || !unit

package compute

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type ComputeSuite struct {
	suite.Suite
	node             *node.Compute
	config           node.ComputeConfig
	cm               *system.CleanupManager
	executor         *noop_executor.NoopExecutor
	publisher        *noop_publisher.NoopPublisher
	stateResolver    resolver.StateResolver
	bidChannel       chan compute.BidResult
	failureChannel   chan compute.ComputeError
	completedChannel chan compute.RunResult
}

func (s *ComputeSuite) SetupTest() {
	s.setupConfig()
	s.setupNode()
}

// setupConfig creates a new config for testing
func (s *ComputeSuite) setupConfig() {
	storePath := filepath.Join(s.T().TempDir(), fmt.Sprint(time.Now().UnixNano()))
	s.Require().NoError(os.MkdirAll(storePath, 0755))

	executionStore, err := boltdb.NewStore(context.Background(), filepath.Join(storePath, "executions.db"))
	s.Require().NoError(err)

	cfg, err := node.NewComputeConfigWith(node.ComputeConfigParams{
		TotalResourceLimits: models.Resources{
			CPU: 2,
		},
		ExecutionStore: executionStore,
	})
	s.Require().NoError(err)
	s.config = cfg
}

func (s *ComputeSuite) setupNode() {
	ctx := context.Background()
	s.cm = system.NewCleanupManager()
	s.T().Cleanup(func() { s.cm.Cleanup(ctx) })

	s.executor = noop_executor.NewNoopExecutor()
	s.publisher = noop_publisher.NewNoopPublisher()
	s.bidChannel = make(chan compute.BidResult)
	s.completedChannel = make(chan compute.RunResult)
	s.failureChannel = make(chan compute.ComputeError)

	apiServer, err := publicapi.NewAPIServer(publicapi.ServerParams{
		Router:     echo.New(),
		Address:    "0.0.0.0",
		Port:       0,
		Config:     publicapi.DefaultConfig(),
		Authorizer: authz.AlwaysAllow,
	})
	s.NoError(err)

	storagePath := s.T().TempDir()

	noopstorage := noop_storage.NewNoopStorage()
	callback := compute.CallbackMock{
		OnBidCompleteHandler: func(ctx context.Context, result compute.BidResult) {
			s.bidChannel <- result
		},
		OnRunCompleteHandler: func(ctx context.Context, result compute.RunResult) {
			s.completedChannel <- result
		},
		OnComputeFailureHandler: func(ctx context.Context, err compute.ComputeError) {
			s.failureChannel <- err
		},
	}

	s.node, err = node.NewComputeNode(
		context.Background(),
		"test",
		s.cm,
		apiServer,
		s.config,
		storagePath,
		provider.NewNoopProvider[storage.Storage](noopstorage),
		provider.NewNoopProvider[executor.Executor](s.executor),
		provider.NewNoopProvider[publisher.Publisher](s.publisher),
		callback,
	)
	s.NoError(err)
	s.stateResolver = *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})

	s.T().Cleanup(func() { close(s.bidChannel) })
}

func (s *ComputeSuite) askForBid(ctx context.Context, execution *models.Execution) compute.BidResult {
	_, err := s.node.LocalEndpoint.AskForBid(ctx, compute.AskForBidRequest{
		RoutingMetadata: compute.RoutingMetadata{
			TargetPeerID: s.node.ID,
			SourcePeerID: s.node.ID,
		},
		Execution:       execution,
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

func (s *ComputeSuite) prepareAndAskForBid(ctx context.Context, execution *models.Execution) string {
	result := s.askForBid(ctx, execution)
	s.True(result.Accepted)
	return result.ExecutionID
}

func (s *ComputeSuite) prepareAndRun(ctx context.Context, execution *models.Execution) string {
	executionID := s.prepareAndAskForBid(ctx, execution)

	// run the job
	_, err := s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCompleted))
	s.NoError(err)
	return executionID
}
