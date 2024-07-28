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
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
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
	_, err := s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)

	// Wait for the execution to start
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateRunning))
	s.NoError(err)

	// Cancel the execution
	_, err = s.node.LocalEndpoint.CancelExecution(ctx, compute.CancelExecutionRequest{
		ExecutionID: executionID,
	})

	// Wait for the execution to be cancelled
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCancelled))
	s.NoError(err)
}

func (s *CancelSuite) TestCancelDocker() {
	docker.MustHaveDocker(s.T())
	ctx := context.Background()

	// prepare a docker execution that sleeps for 10 seconds so we can cancel it
	dockerSpec, err := dockermodels.NewDockerEngineBuilder("ubuntu").
		WithEntrypoint("bash", "-c", "sleep 10").
		Build()
	s.NoError(err)

	execution := mock.Execution()
	execution.Job.Task().Engine = dockerSpec

	// Create and submit the execution
	executionID := s.prepareAndAskForBid(ctx, execution)
	_, err = s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)

	// Wait for the execution to start
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateRunning))
	s.NoError(err)

	// We need to wait for the container to become active, before we cancel the execution.
	time.Sleep(time.Second * 1)
	_, err = s.node.LocalEndpoint.CancelExecution(ctx, compute.CancelExecutionRequest{
		ExecutionID: executionID,
	})

	// Wait for the execution to be cancelled
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCancelled))
	s.NoError(err)

}

func (s *CancelSuite) TestDoesntExist() {
	ctx := context.Background()
	_, err := s.node.LocalEndpoint.CancelExecution(ctx, compute.CancelExecutionRequest{ExecutionID: uuid.NewString()})
	s.Error(err)
}

func (s *CancelSuite) TestStates() {
	ctx := context.Background()

	for _, tc := range []struct {
		state      store.LocalExecutionStateType
		shouldFail bool
	}{
		// These states should allow the execution to be cancelled
		{store.ExecutionStateCreated, false},
		{store.ExecutionStateBidAccepted, false},
		{store.ExecutionStateRunning, false},
		{store.ExecutionStatePublishing, false},

		// These states should not allow the execution to be cancelled
		{store.ExecutionStateCancelled, true},
		{store.ExecutionStateCompleted, true},
		{store.ExecutionStateCancelled, true},
	} {
		s.Run(tc.state.String(), func() {
			executionID := s.prepareAndAskForBid(ctx, mock.Execution())
			err := s.node.ExecutionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
				ExecutionID: executionID,
				NewState:    tc.state,
			})
			s.NoError(err)

			_, err = s.node.LocalEndpoint.CancelExecution(ctx, compute.CancelExecutionRequest{ExecutionID: executionID})
			if tc.shouldFail {
				// verify error and state is still the same
				s.Error(err)
				state, err := s.node.ExecutionStore.GetExecution(ctx, executionID)
				s.NoError(err)
				s.Equal(tc.state, state.State)
			} else {
				s.NoError(err)
				state, err := s.node.ExecutionStore.GetExecution(ctx, executionID)
				s.NoError(err)
				s.Equal(store.ExecutionStateCancelled, state.State)
			}

		})
	}
}
