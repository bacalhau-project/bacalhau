//go:build integration || !unit

package compute

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
)

type CancelSuite struct {
	ComputeSuite
}

func TestCancelSuite(t *testing.T) {
	suite.Run(t, new(CancelSuite))
}

func (s *CancelSuite) TestCancel() {
	ctx := context.Background()

	// Set the delay to 5 seconds, so we can cancel the execution
	s.executor.Config.ExternalHooks.JobHandler = noop_executor.DelayedJobHandler(5 * time.Second)

	// Create and submit the execution
	executionID := s.prepareAndAskForBid(ctx, mock.Execution())
	_, err := s.node.LocalEndpoint.BidAccepted(ctx, messages.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)

	// Wait for the execution to start
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(models.ExecutionStateRunning))
	s.NoError(err)

	// Cancel the execution
	_, err = s.node.LocalEndpoint.CancelExecution(ctx, messages.CancelExecutionRequest{
		ExecutionID: executionID,
	})

	// Wait for the execution to be cancelled
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(models.ExecutionStateCancelled))
	s.NoError(err)
}

func (s *CancelSuite) TestCancelDocker() {
	docker.MustHaveDocker(s.T())
	ctx := context.Background()

	// prepare a docker execution that sleeps for 10 seconds so we can cancel it
	dockerSpec, err := dockermodels.NewDockerEngineBuilder("busybox:1.37.0").
		WithEntrypoint("sh", "-c", "sleep 10").
		Build()
	s.NoError(err)

	execution := mock.Execution()
	execution.Job.Task().Engine = dockerSpec

	// Create and submit the execution
	executionID := s.prepareAndAskForBid(ctx, execution)
	_, err = s.node.LocalEndpoint.BidAccepted(ctx, messages.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)

	// Wait for the execution to start
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(models.ExecutionStateRunning))
	s.NoError(err)

	// We need to wait for the container to become active, before we cancel the execution.
	time.Sleep(time.Second * 1)
	_, err = s.node.LocalEndpoint.CancelExecution(ctx, messages.CancelExecutionRequest{
		ExecutionID: executionID,
	})

	// Wait for the execution to be cancelled
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(models.ExecutionStateCancelled))
	s.NoError(err)

}

func (s *CancelSuite) TestDoesntExist() {
	ctx := context.Background()
	_, err := s.node.LocalEndpoint.CancelExecution(ctx, messages.CancelExecutionRequest{ExecutionID: uuid.NewString()})
	s.Error(err)
}

func (s *CancelSuite) TestStates() {
	ctx := context.Background()

	for _, tc := range []struct {
		state      models.ExecutionStateType
		shouldFail bool
	}{
		// These states should allow the execution to be cancelled
		{models.ExecutionStateNew, false},
		{models.ExecutionStateBidAccepted, false},
		{models.ExecutionStateRunning, false},
		{models.ExecutionStatePublishing, false},

		// These states should not allow the execution to be cancelled
		{models.ExecutionStateCancelled, true},
		{models.ExecutionStateCompleted, true},
		{models.ExecutionStateCancelled, true},
	} {
		s.Run(tc.state.String(), func() {
			s.TearDownTest()
			s.SetupTest()
			executionID := s.prepareAndAskForBid(ctx, mock.Execution())

			// stop the watchers so execution state doesn't get updated beyond what we set in the test
			s.Require().NoError(s.node.Watchers.Stop(ctx))

			err := s.node.ExecutionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
				ExecutionID: executionID,
				NewValues:   models.Execution{ComputeState: models.NewExecutionState(tc.state)},
			})
			s.NoError(err)

			_, err = s.node.LocalEndpoint.CancelExecution(ctx, messages.CancelExecutionRequest{ExecutionID: executionID})
			if tc.shouldFail {
				// verify error and state is still the same
				s.Error(err)
				execution, err := s.node.ExecutionStore.GetExecution(ctx, executionID)
				s.NoError(err)
				s.Equal(tc.state, execution.ComputeState.StateType)
			} else {
				s.NoError(err)
				execution, err := s.node.ExecutionStore.GetExecution(ctx, executionID)
				s.NoError(err)
				s.Equal(models.ExecutionStateCancelled, execution.ComputeState.StateType)
			}

		})
	}
}
