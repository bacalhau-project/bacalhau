//go:build integration || !unit

package compute

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	noop_publisher "github.com/bacalhau-project/bacalhau/pkg/publisher/noop"
	noop_storage "github.com/bacalhau-project/bacalhau/pkg/storage/noop"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	executor_common "github.com/bacalhau-project/bacalhau/pkg/executor"
	dockerexecutor "github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

type CancelDockerExecutionSuite struct {
	suite.Suite
	node             *node.Compute
	store            store.ExecutionStore
	callbackStore    *CallbackStore
	bidChannel       chan compute.BidResult
	failureChannel   chan compute.ComputeError
	completedChannel chan compute.RunResult
}

func TestCancelDockerExecutionSuite(t *testing.T) {
	suite.Run(t, new(CancelDockerExecutionSuite))
}

func (s *CancelDockerExecutionSuite) TestCancelledDockerExeuction() {
	docker.MustHaveDocker(s.T())
	ctx := context.Background()

	_, c := setup.SetupBacalhauRepoForTesting(s.T())

	executionStore, err := boltdb.NewStore(context.Background(), filepath.Join(s.T().TempDir(), "executions.db"))
	s.store = executionStore
	s.Require().NoError(err)

	s.callbackStore = &CallbackStore{}
	s.callbackStore.GetExecutionFn = s.store.GetExecution
	s.callbackStore.GetExecutionsFn = s.store.GetExecutions
	s.callbackStore.GetLiveExecutionsFn = s.store.GetLiveExecutions
	s.callbackStore.GetExecutionHistoryFn = s.store.GetExecutionHistory
	s.callbackStore.CreateExecutionFn = s.store.CreateExecution
	s.callbackStore.DeleteExecutionFn = s.store.DeleteExecution
	s.callbackStore.GetExecutionCountFn = s.store.GetExecutionCount
	s.callbackStore.CloseFn = s.store.Close

	// This test we are testing that when an execution is cancelled, the executor, no longer
	// tries to mark the execution as completed. The sync group here is 3, mainly because
	// the execution state will go to
	// 1. ExecutionStateBidAccepted
	// 2. ExecutionStateRunning
	// 3. ExecutionStateCancelled
	// If any more requests come in to update execution state, the wait group would go negative indicating some other state update is
	// being called to.
	exeuctionStateUpdateWg := sync.WaitGroup{}
	exeuctionStateUpdateWg.Add(3)
	s.callbackStore.UpdateExecutionStateFn = func(ctx context.Context, request store.UpdateExecutionStateRequest) error {
		defer exeuctionStateUpdateWg.Done()
		return s.store.UpdateExecutionState(ctx, request)
	}

	computeConfig, err := node.NewComputeConfigWith(c.Node.ComputeStoragePath, node.ComputeConfigParams{
		TotalResourceLimits: models.Resources{
			CPU: 2,
		},
		ExecutionStore: s.callbackStore,
	})
	s.Require().NoError(err)

	dockerExecutor, err := dockerexecutor.NewExecutor("compute-node-docker-executor-test", configenv.Testing.Node.Compute.ManifestCache)
	s.Require().NoError(err)

	apiServer, err := publicapi.NewAPIServer(publicapi.ServerParams{
		Router:     echo.New(),
		Address:    "0.0.0.0",
		Port:       0,
		Config:     publicapi.DefaultConfig(),
		Authorizer: authz.AlwaysAllow,
	})

	s.NoError(err)

	storagePath := s.T().TempDir()
	repoPath := s.T().TempDir()
	s.bidChannel = make(chan compute.BidResult, 1)
	s.completedChannel = make(chan compute.RunResult)
	s.failureChannel = make(chan compute.ComputeError)

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
		apiServer,
		computeConfig,
		storagePath,
		repoPath,
		provider.NewNoopProvider[storage.Storage](noopstorage),
		provider.NewMappedProvider(map[string]executor_common.Executor{
			models.EngineDocker: dockerExecutor,
		}),
		provider.NewNoopProvider[publisher.Publisher](noop_publisher.NewNoopPublisher()),
		callback,
		ManagementEndpointMock{},
		map[string]string{},
		HeartbeatClientMock{},
	)

	s.Require().NoError(err)
	stateResolver := *resolver.NewStateResolver(resolver.StateResolverParams{
		ExecutionStore: s.node.ExecutionStore,
	})

	es, err := dockermodels.NewDockerEngineBuilder("ubuntu").
		WithEntrypoint("bash", "-c", "sleep 10").
		Build()

	s.Require().NoError(err)
	s.Require().NoError(err)
	task := mock.TaskBuilder().
		Engine(es).
		BuildOrDie()
	job := mock.Job()
	job.Tasks[0] = task
	job.Normalize()

	execution := mock.ExecutionForJob(job)
	executionID := s.prepareAndAskForBid(ctx, execution)

	_, err = s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
	s.Require().NoError(err)

	err = stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateBidAccepted))
	s.NoError(err)

	err = stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateRunning))
	s.NoError(err)

	// We need to wait for the container to become active, before we cancel the execution.
	time.Sleep(time.Second * 2)
	_, err = s.node.LocalEndpoint.CancelExecution(ctx, compute.CancelExecutionRequest{
		ExecutionID: executionID,
	})
	s.NoError(err)
	err = stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCancelled))
	s.NoError(err)
	exeuctionStateUpdateWg.Wait()
}

func (s *CancelDockerExecutionSuite) askForBid(ctx context.Context, execution *models.Execution) compute.BidResult {
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

func (s *CancelDockerExecutionSuite) prepareAndAskForBid(ctx context.Context, execution *models.Execution) string {
	result := s.askForBid(ctx, execution)
	s.True(result.Accepted)
	return result.ExecutionID
}
