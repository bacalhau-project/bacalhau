//go:build integration || !unit

package compute

import (
	"context"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	executor_common "github.com/bacalhau-project/bacalhau/pkg/executor"
	dockerexecutor "github.com/bacalhau-project/bacalhau/pkg/executor/docker"
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
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type ComputeSuite struct {
	suite.Suite
	node             *node.Compute
	config           node.NodeConfig
	capacity         models.Resources
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
	capacityConfig := models.ResourcesConfig{
		CPU: "2",
	}
	capacity, err := capacityConfig.ToResources()
	s.Require().NoError(err)

	s.capacity = *capacity
	bacalhauConfig, err := config.NewTestConfigWithOverrides(types.Bacalhau{
		Compute: types.Compute{
			AllocatedCapacity: types.ResourceScalerFromModelsResourceConfig(capacityConfig),
		},
	})
	s.Require().NoError(err)

	nodeID := "test"
	s.executor = noop_executor.NewNoopExecutor()
	s.publisher = noop_publisher.NewNoopPublisher()
	s.completedChannel = make(chan compute.RunResult)
	s.failureChannel = make(chan compute.ComputeError)

	dockerExecutor, err := dockerexecutor.NewExecutor(nodeID, bacalhauConfig.Engines.Types.Docker)

	s.config = node.NodeConfig{
		NodeID:         nodeID,
		BacalhauConfig: bacalhauConfig,
		SystemConfig:   node.DefaultSystemConfig(),
		DependencyInjector: node.NodeDependencyInjector{
			StorageProvidersFactory: node.StorageProvidersFactoryFunc(
				func(ctx context.Context, nodeConfig node.NodeConfig) (storage.StorageProvider, error) {
					return provider.NewNoopProvider[storage.Storage](noop_storage.NewNoopStorage()), nil
				}),
			ExecutorsFactory: node.ExecutorsFactoryFunc(
				func(ctx context.Context, nodeConfig node.NodeConfig) (executor_common.ExecProvider, error) {
					return provider.NewMappedProvider(map[string]executor_common.Executor{
						models.EngineNoop:   s.executor,
						models.EngineDocker: dockerExecutor,
					}), nil
				}),
			PublishersFactory: node.PublishersFactoryFunc(
				func(ctx context.Context, nodeConfig node.NodeConfig) (publisher.PublisherProvider, error) {
					return provider.NewNoopProvider[publisher.Publisher](s.publisher), nil
				}),
		},
	}

	s.T().Cleanup(func() { os.RemoveAll(bacalhauConfig.DataDir) })
}

func (s *ComputeSuite) setupNode() {
	ctx := context.Background()
	s.cm = system.NewCleanupManager()
	s.T().Cleanup(func() { s.cm.Cleanup(ctx) })

	s.bidChannel = make(chan compute.BidResult, 1)
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

	// setup nats server and client
	ns, nc := testutils.StartNats(s.T())
	s.T().Cleanup(func() { nc.Close() })
	s.T().Cleanup(func() { ns.Shutdown() })

	messageSerDeRegistry, err := node.CreateMessageSerDeRegistry()
	s.Require().NoError(err)

	// create the compute node
	s.node, err = node.NewComputeNode(
		ctx,
		s.config,
		apiServer,
		nc,
		callback,
		ManagementEndpointMock{},
		messageSerDeRegistry,
	)
	s.NoError(err)
	s.stateResolver = *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})

	s.T().Cleanup(func() { close(s.bidChannel) })
	s.T().Cleanup(func() { s.node.Cleanup(ctx) })
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
