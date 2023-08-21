//go:build unit || !integration

package inmemory

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	executionStore      store.ExecutionStore
	localExecutionState store.LocalExecutionState
	execution           *models.Execution
}

func (s *Suite) SetupTest() {
	s.executionStore = NewStore()
	s.execution = mock.ExecutionForJob(mock.Job())
	s.localExecutionState = *store.NewLocalExecutionState(s.execution, "nodeID-1")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestCreateExecution() {
	err := s.executionStore.CreateExecution(context.Background(), s.localExecutionState)
	s.NoError(err)

	// verify the execution was created
	readExecution, err := s.executionStore.GetExecution(context.Background(), s.execution.ID)
	s.NoError(err)
	s.Equal(s.localExecutionState, readExecution)

	// verify a history entry was created
	history, err := s.executionStore.GetExecutionHistory(context.Background(), s.execution.ID)
	s.NoError(err)
	s.Len(history, 1)
	s.verifyHistory(history[0], readExecution, store.ExecutionStateUndefined, newExecutionComment)
}

func (s *Suite) TestCreateExecution_AlreadyExists() {
	err := s.executionStore.CreateExecution(context.Background(), s.localExecutionState)
	s.NoError(err)

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
	s.NoError(err)

	readExecutions, err := s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.NoError(err)
	s.Len(readExecutions, 1)
	s.Equal(s.localExecutionState, readExecutions[0])

	// Create another execution for the same job
	anotherExecution := mock.ExecutionForJob(s.execution.Job)
	anotherExecutionState := *store.NewLocalExecutionState(anotherExecution, "nodeID")
	err = s.executionStore.CreateExecution(ctx, anotherExecutionState)
	s.NoError(err)

	readExecutions, err = s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.NoError(err)
	s.Len(readExecutions, 2)
	s.Equal(s.localExecutionState, readExecutions[0])
	s.Equal(anotherExecutionState, readExecutions[1])
}

func (s *Suite) TestGetExecutions_DoesntExist() {
	_, err := s.executionStore.GetExecutions(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestUpdateExecution() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID: s.execution.ID,
		NewState:    store.ExecutionStatePublishing,
		Comment:     "Hello There!",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.NoError(err)

	// verify the update happened as expected
	readExecution, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	s.NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.localExecutionState.Version+1, readExecution.Version)

	// verify a new history entry was created
	history, err := s.executionStore.GetExecutionHistory(ctx, s.execution.ID)
	s.NoError(err)
	s.Len(history, 2)
	s.verifyHistory(history[1], readExecution, s.localExecutionState.State, request.Comment)
}

func (s *Suite) TestUpdateExecution_ConditionsPass() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedState:   s.localExecutionState.State,
		ExpectedVersion: s.localExecutionState.Version,
		NewState:        store.ExecutionStatePublishing,
		Comment:         "Hello There!",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.NoError(err)

	// verify the update happened as expected
	readExecution, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	s.NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.localExecutionState.Version+1, readExecution.Version)
}

func (s *Suite) TestUpdateExecution_ConditionsStateFail() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:   s.execution.ID,
		ExpectedState: store.ExecutionStateBidAccepted,
		NewState:      store.ExecutionStatePublishing,
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.ErrorAs(err, &store.ErrInvalidExecutionState{})
}

func (s *Suite) TestUpdateExecution_ConditionsVersionFail() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.NoError(err)

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
	s.NoError(err)

	err = s.executionStore.DeleteExecution(context.Background(), s.execution.ID)
	s.NoError(err)

	_, err = s.executionStore.GetExecution(context.Background(), s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})

	_, err = s.executionStore.GetExecutions(context.Background(), s.execution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestDeleteExecution_MultiEntries() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.localExecutionState)
	s.NoError(err)

	// second execution with same jobID
	secondExecution := mock.ExecutionForJob(s.execution.Job)
	secondExecutionState := *store.NewLocalExecutionState(secondExecution, "nodeID")
	err = s.executionStore.CreateExecution(ctx, secondExecutionState)

	// third execution with different jobID
	thirdExecution := mock.ExecutionForJob(mock.Job())
	thirdExecutionState := *store.NewLocalExecutionState(thirdExecution, "nodeID")
	err = s.executionStore.CreateExecution(ctx, thirdExecutionState)
	s.NoError(err)

	// validate pre-state
	firstJobExecutions, err := s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.NoError(err)
	s.Len(firstJobExecutions, 2)

	secondJobExecutions, err := s.executionStore.GetExecutions(ctx, thirdExecution.JobID)
	s.NoError(err)
	s.Len(secondJobExecutions, 1)

	// delete first execution
	err = s.executionStore.DeleteExecution(ctx, s.execution.ID)
	s.NoError(err)
	_, err = s.executionStore.GetExecution(ctx, s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	executions, err := s.executionStore.GetExecutions(ctx, s.execution.JobID)
	s.NoError(err)
	s.Len(executions, 1)

	// delete second execution
	err = s.executionStore.DeleteExecution(ctx, secondExecution.ID)
	s.NoError(err)
	_, err = s.executionStore.GetExecution(ctx, secondExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	executions, err = s.executionStore.GetExecutions(ctx, secondExecution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})

	// delete third execution
	err = s.executionStore.DeleteExecution(ctx, thirdExecution.ID)
	s.NoError(err)
	_, err = s.executionStore.GetExecution(ctx, thirdExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	_, err = s.executionStore.GetExecutions(ctx, thirdExecution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestDeleteExecution_DoesntExist() {
	err := s.executionStore.DeleteExecution(context.Background(), uuid.NewString())
	s.NoError(err)
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
