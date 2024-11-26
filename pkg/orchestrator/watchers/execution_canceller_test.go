//go:build unit || !integration

package watchers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type ExecutionCancellerTestSuite struct {
	suite.Suite
	ctx       context.Context
	ctrl      *gomock.Controller
	jobStore  *jobstore.MockStore
	canceller *ExecutionCanceller
}

func TestExecutionCancellerTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutionCancellerTestSuite))
}

func (s *ExecutionCancellerTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.ctrl = gomock.NewController(s.T())
	s.jobStore = jobstore.NewMockStore(s.ctrl)
	s.canceller = NewExecutionCanceller(s.jobStore)
}

func (s *ExecutionCancellerTestSuite) TestHandleEvent_InvalidObject() {
	err := s.canceller.HandleEvent(s.ctx, watcher.Event{
		Object: "not an execution upsert",
	})
	s.Error(err)
}

func (s *ExecutionCancellerTestSuite) TestHandleEvent_NoStateChange() {
	// Create upsert with identical Previous and Current
	execution := mock.Execution()
	upsert := models.ExecutionUpsert{
		Previous: execution,
		Current:  execution,
	}

	// No jobstore updates should happen
	err := s.canceller.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *ExecutionCancellerTestSuite) TestHandleEvent_NonCancellationTransition() {
	// Create state transition that isn't a cancellation
	upsert := setupStateTransition(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateNew,
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateNew,
	)

	// No jobstore updates should happen
	err := s.canceller.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *ExecutionCancellerTestSuite) TestHandleEvent_CancellationTransition() {
	// Create state transition for cancellation
	upsert := setupStateTransition(
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateRunning,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateRunning,
	)

	// Expect jobstore update to mark execution as cancelled
	s.jobStore.EXPECT().UpdateExecution(s.ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, req jobstore.UpdateExecutionRequest) error {
			s.Equal(upsert.Current.ID, req.ExecutionID)
			s.Equal(models.ExecutionStateCancelled, req.NewValues.ComputeState.StateType)
			return nil
		})

	err := s.canceller.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}

func (s *ExecutionCancellerTestSuite) TestHandleEvent_JobStoreError() {
	// Create state transition for cancellation
	upsert := setupStateTransition(
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateRunning,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateRunning,
	)

	// Simulate jobstore error
	s.jobStore.EXPECT().UpdateExecution(s.ctx, gomock.Any()).Return(
		bacerrors.New("failed to update execution"))

	// Should not return error even if jobstore update fails
	err := s.canceller.HandleEvent(s.ctx, createExecutionEvent(upsert))
	s.NoError(err)
}
