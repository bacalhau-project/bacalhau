//go:build unit || !integration

package boltdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	ctx                 context.Context
	executionStore      store.ExecutionStore
	localExecutionState store.LocalExecutionState
	execution           *models.Execution
	dbFile              string
}

func (s *Suite) SetupTest() {
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-executionstore")
	s.dbFile = filepath.Join(dir, "test.boltdb")
	s.executionStore, _ = NewStore(s.ctx, s.dbFile)
	s.execution = mock.ExecutionForJob(mock.Job())
	s.localExecutionState = *store.NewLocalExecutionState(s.execution, "nodeID-1")
}

func (s *Suite) TearDownTest() {
	os.Remove(s.dbFile)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestCreateExecution() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	// verify the execution was created
	readExecution, err := s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Equal(s.localExecutionState, readExecution)

	// verify a history entry was created
	history, err := s.executionStore.GetExecutionHistory(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Len(history, 1)
	s.verifyHistory(history[0], readExecution, store.ExecutionStateUndefined, newExecutionComment)
}

func (s *Suite) TestCreateExecution_AlreadyExists() {
	err := s.executionStore.CreateExecution(context.Background(), s.localExecutionState)
	s.Require().NoError(err)

	err = s.executionStore.CreateExecution(context.Background(), s.localExecutionState)
	s.Error(err)
}

func (s *Suite) TestCreateExecution_InvalidState() {
	s.localExecutionState.State = store.ExecutionStateRunning
	err := s.executionStore.CreateExecution(context.Background(), s.localExecutionState)
	s.Error(err)
}

func (s *Suite) TestGetExecution_DoesntExist() {
	_, err := s.executionStore.GetExecution(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
}

func (s *Suite) TestGetExecutions() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.Require().NoError(err)

	readExecutions, err := s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(readExecutions, 1)
	s.Equal(s.localExecutionState, readExecutions[0])

	// Create another execution for the same job
	anotherExecution := mock.ExecutionForJob(s.execution.Job)
	anotherExecutionState := *store.NewLocalExecutionState(anotherExecution, "nodeID")
	err = s.executionStore.CreateExecution(ctx, anotherExecutionState)
	s.Require().NoError(err)

	readExecutions, err = s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(readExecutions, 2)
	s.Equal(s.localExecutionState, readExecutions[0])
	s.Equal(anotherExecutionState, readExecutions[1])
}

func (s *Suite) TestGetLiveExecutions() {
	ctx := context.Background()

	localExec := store.NewLocalExecutionState(s.execution, "req1")
	err := s.executionStore.CreateExecution(ctx, *localExec)
	s.Require().NoError(err)

	err = s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: s.execution.ID,
		NewState:    store.ExecutionStateRunning,
	})
	s.Require().NoError(err)

	execs, err := s.executionStore.GetLiveExecutions(ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, len(execs))
	s.Require().Equal(s.execution.ID, execs[0].Execution.ID)
}

func (s *Suite) TestFullLiveExecutions() {
	ctx := context.Background()

	requireLive := func(executionID string) {
		execs, err := s.executionStore.GetLiveExecutions(ctx)
		s.Require().NoError(err)
		s.Require().Equal(1, len(execs))
		s.Require().Equal(executionID, execs[0].Execution.ID)
	}
	requireNotLive := func(executionID string) {
		execs, err := s.executionStore.GetLiveExecutions(ctx)
		s.Require().NoError(err)
		s.Require().Equal([]store.LocalExecutionState{}, execs)
	}

	localExec := store.NewLocalExecutionState(s.execution, "req1")
	err := s.executionStore.CreateExecution(ctx, *localExec)
	s.Require().NoError(err)

	type testdata struct {
		state       store.LocalExecutionStateType
		requirement func(string)
	}
	testcases := []testdata{
		{store.ExecutionStateBidAccepted, requireNotLive},
		{store.ExecutionStateCreated, requireNotLive},
		{store.ExecutionStateRunning, requireLive},
		{store.ExecutionStateCancelled, requireNotLive},
	}

	for _, tc := range testcases {
		s.T().Run(tc.state.String(), func(t *testing.T) {
			err = s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
				ExecutionID: s.execution.ID,
				NewState:    tc.state,
			})
			s.Require().NoError(err)

			tc.requirement(s.execution.ID)
		})
	}

}

func (s *Suite) TestGetMultipleLiveExecutions() {
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		exec := mock.ExecutionForJob(mock.Job())
		exec.ID = fmt.Sprintf("%d", i+1)
		localExec := store.NewLocalExecutionState(exec, "req1")
		err := s.executionStore.CreateExecution(ctx, *localExec)
		s.Require().NoError(err)

		err = s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID: exec.ID,
			NewState:    store.ExecutionStateRunning,
		})
		s.Require().NoError(err)
	}

	execs, err := s.executionStore.GetLiveExecutions(ctx)
	s.Require().NoError(err)
	s.Require().Equal(3, len(execs))

	// We want to make sure the executions are returned with increasing update times
	// that is, oldest first. On windows, the Update times are the same and so we
	// allow them to compare as <=.
	s.Require().LessOrEqual(execs[0].UpdateTime, execs[1].UpdateTime)
	s.Require().LessOrEqual(execs[1].UpdateTime, execs[2].UpdateTime)
}

func (s *Suite) TestGetExecutions_DoesntExist() {
	_, err := s.executionStore.GetExecutions(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestUpdateExecution() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID: s.execution.ID,
		NewState:    store.ExecutionStatePublishing,
		Comment:     "Hello There!",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.Require().NoError(err)

	// verify the update happened as expected
	readExecution, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.localExecutionState.Version+1, readExecution.Version)

	// verify a new history entry was created
	history, err := s.executionStore.GetExecutionHistory(ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Len(history, 2)
	s.verifyHistory(history[1], readExecution, s.localExecutionState.State, request.Comment)
}

func (s *Suite) TestUpdateExecution_ConditionsPass() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedStates:  []store.LocalExecutionStateType{s.localExecutionState.State},
		ExpectedVersion: s.localExecutionState.Version,
		NewState:        store.ExecutionStatePublishing,
		Comment:         "Hello There!",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.Require().NoError(err)

	// verify the update happened as expected
	readExecution, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.localExecutionState.Version+1, readExecution.Version)
}

func (s *Suite) TestGetExecutionCount() {
	ctx := context.Background()

	states := []store.LocalExecutionStateType{
		store.ExecutionStateBidAccepted,
		store.ExecutionStateBidAccepted,
		store.ExecutionStateBidAccepted,
		store.ExecutionStateCompleted,
		store.ExecutionStateCompleted,
	}

	for _, state := range states {
		execution := mock.ExecutionForJob(mock.Job())
		executionState := *store.NewLocalExecutionState(execution, "nodeID")
		err := s.executionStore.CreateExecution(ctx, executionState)
		s.Require().NoError(err)

		request := store.UpdateExecutionStateRequest{
			ExecutionID:     execution.ID,
			ExpectedStates:  []store.LocalExecutionStateType{executionState.State},
			ExpectedVersion: executionState.Version,
			NewState:        state,
			Comment:         "Hello There!",
		}

		err = s.executionStore.UpdateExecutionState(ctx, request)
		s.Require().NoError(err)
	}

	// Close and re-open the execution store so we can
	// re-populate the counter
	s.executionStore.Close(s.ctx)
	s.executionStore, _ = NewStore(s.ctx, s.dbFile)

	c, err := s.executionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	s.Require().NoError(err)
	s.Equal(uint64(2), c)
}

func (s *Suite) TestStateCounterChange() {
	ctx := context.Background()

	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.Require().NoError(err)

	request := store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedStates:  []store.LocalExecutionStateType{s.localExecutionState.State},
		ExpectedVersion: s.localExecutionState.Version,
		NewState:        store.ExecutionStateBidAccepted,
		Comment:         "Starting",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.Require().NoError(err)

	accepted, err := s.executionStore.GetExecutionCount(ctx, store.ExecutionStateBidAccepted)
	s.Require().NoError(err)
	completed, err := s.executionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	s.Require().NoError(err)
	s.Equal(uint64(1), accepted)
	s.Equal(uint64(0), completed)

	request = store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedStates:  []store.LocalExecutionStateType{store.ExecutionStateBidAccepted},
		ExpectedVersion: s.localExecutionState.Version + 1,
		NewState:        store.ExecutionStateCompleted,
		Comment:         "Completed",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.Require().NoError(err)

	accepted, err = s.executionStore.GetExecutionCount(ctx, store.ExecutionStateBidAccepted)
	s.Require().NoError(err)
	completed, err = s.executionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	s.Require().NoError(err)
	s.Equal(uint64(0), accepted)
	s.Equal(uint64(1), completed)

}

func (s *Suite) TestUpdateExecution_ConditionsStateFail() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:    s.execution.ID,
		ExpectedStates: []store.LocalExecutionStateType{store.ExecutionStateBidAccepted},
		NewState:       store.ExecutionStatePublishing,
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.ErrorAs(err, &store.ErrInvalidExecutionState{})
}

func (s *Suite) TestUpdateExecution_ConditionsVersionFail() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedVersion: s.localExecutionState.Version + 99,
		NewState:        store.ExecutionStatePublishing,
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.ErrorAs(err, &store.ErrInvalidExecutionVersion{})
}

func (s *Suite) TestDeleteExecution() {
	err := s.executionStore.CreateExecution(context.Background(), s.localExecutionState)
	s.Require().NoError(err)

	err = s.executionStore.DeleteExecution(context.Background(), s.execution.ID)
	s.Require().NoError(err)

	_, err = s.executionStore.GetExecution(context.Background(), s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})

	_, err = s.executionStore.GetExecutions(context.Background(), s.execution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestDeleteExecution_MultiEntries() {
	ctx := s.ctx
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.Require().NoError(err)

	// second execution with same jobID
	secondExecution := mock.ExecutionForJob(s.execution.Job)
	secondExecutionState := *store.NewLocalExecutionState(secondExecution, "nodeID")
	err = s.executionStore.CreateExecution(ctx, secondExecutionState)
	require.NoError(s.T(), err)

	// third execution with different jobID
	thirdExecution := mock.ExecutionForJob(mock.Job())
	thirdExecutionState := *store.NewLocalExecutionState(thirdExecution, "nodeID")
	err = s.executionStore.CreateExecution(ctx, thirdExecutionState)
	s.Require().NoError(err)

	// validate pre-state
	firstJobExecutions, err := s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(firstJobExecutions, 2)

	secondJobExecutions, err := s.executionStore.GetExecutions(ctx, thirdExecution.JobID)
	s.Require().NoError(err)
	s.Len(secondJobExecutions, 1)
	// delete first execution
	err = s.executionStore.DeleteExecution(ctx, s.execution.ID)
	s.Require().NoError(err)
	_, err = s.executionStore.GetExecution(ctx, s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	executions, err := s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(executions, 1)

	// delete second execution
	err = s.executionStore.DeleteExecution(ctx, secondExecution.ID)
	s.Require().NoError(err)
	_, err = s.executionStore.GetExecution(ctx, secondExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	_, err = s.executionStore.GetExecutions(ctx, secondExecution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})

	// delete third execution
	err = s.executionStore.DeleteExecution(ctx, thirdExecution.ID)
	s.Require().NoError(err)
	_, err = s.executionStore.GetExecution(ctx, thirdExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	_, err = s.executionStore.GetExecutions(ctx, thirdExecution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestDeleteExecution_DoesntExist() {
	err := s.executionStore.DeleteExecution(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
}

func (s *Suite) TestGetExecutionHistory_DoesntExist() {
	_, err := s.executionStore.GetExecutionHistory(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionHistoryNotFound{})
}

func (s *Suite) verifyHistory(history store.LocalStateHistory, newExecution store.LocalExecutionState, previousState store.LocalExecutionStateType, comment string) {
	s.Equal(previousState, history.PreviousState)
	s.Equal(newExecution.Execution.ID, history.ExecutionID)
	s.Equal(newExecution.State, history.NewState)
	s.Equal(newExecution.Version, history.NewVersion)
	s.Equal(newExecution.UpdateTime, history.Time)
	s.Equal(comment, history.Comment)
}
