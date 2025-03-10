//go:build integration || !unit

package compute

import (
	"context"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	executor_common "github.com/bacalhau-project/bacalhau/pkg/executor"
	dockerexecutor "github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
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
	bidChannel       chan legacy.BidResult
	failureChannel   chan legacy.ComputeError
	completedChannel chan legacy.RunResult
	natsServer       *server.Server
	natsClient       *nats.Conn
}

func (s *ComputeSuite) SetupTest() {
	s.T().Skip("ENG-402 Un-skip compute test suite")
	s.setupConfig()
	s.setupNode()
}

func (s *ComputeSuite) TearDownTest() {
	ctx := context.Background()

	// Clean up channels
	if s.bidChannel != nil {
		close(s.bidChannel)
	}

	// Clean up node
	if s.node != nil {
		s.node.Cleanup(ctx)
	}

	// Clean up NATS connections
	if s.natsClient != nil {
		s.natsClient.Close()
	}
	if s.natsServer != nil {
		s.natsServer.Shutdown()
	}

	// Clean up system resources
	if s.cm != nil {
		s.cm.Cleanup(ctx)
	}

	// Clean up config data directory
	if s.config.BacalhauConfig.DataDir != "" {
		os.RemoveAll(s.config.BacalhauConfig.DataDir)
	}
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

	dockerExecutor, err := dockerexecutor.NewExecutor(dockerexecutor.ExecutorParams{
		ID:     nodeID,
		Config: bacalhauConfig.Engines.Types.Docker,
	})

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
}

func (s *ComputeSuite) setupNode() {
	ctx := context.Background()
	s.cm = system.NewCleanupManager()

	s.bidChannel = make(chan legacy.BidResult, 10)
	s.completedChannel = make(chan legacy.RunResult, 10)
	s.failureChannel = make(chan legacy.ComputeError, 10)

	apiServer, err := publicapi.NewAPIServer(publicapi.ServerParams{
		Router:     echo.New(),
		Address:    "0.0.0.0",
		Port:       0,
		Config:     publicapi.DefaultConfig(),
		Authorizer: authz.AlwaysAllow,
	})
	s.NoError(err)

	// setup nats server and client
	ns, nc := testutils.StartNats(s.T())
	s.natsServer = ns
	s.natsClient = nc
	clientFactory := func(_ context.Context) (*nats.Conn, error) {
		return nc, nil
	}

	// create the compute node
	s.node, err = node.NewComputeNode(
		ctx,
		s.config,
		apiServer,
		natsutil.ClientFactoryFunc(clientFactory),
		mockInfoProvider{},
	)
	s.NoError(err)
	s.stateResolver = *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})
}

func (s *ComputeSuite) askForBid(ctx context.Context, execution *models.Execution) legacy.BidResult {
	_, err := s.node.LocalEndpoint.AskForBid(ctx, legacy.AskForBidRequest{
		RoutingMetadata: legacy.RoutingMetadata{
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
		return legacy.BidResult{}
	}
}

func (s *ComputeSuite) prepareAndAskForBid(ctx context.Context, execution *models.Execution) string {
	result := s.askForBid(ctx, execution)
	s.True(result.Accepted)
	return result.ExecutionID
}
