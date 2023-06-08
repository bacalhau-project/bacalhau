//go:build unit || !integration

package persistent

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	executionStore store.ExecutionStore
	execution      store.Execution
}

func (s *Suite) SetupTest() {
	s.executionStore, _ = NewStore()
	s.execution = newExecution()

	s.execution.CreateTime = s.execution.CreateTime.Round(0)
	s.execution.UpdateTime = s.execution.UpdateTime.Round(0)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestCreateExecution() {
	err := s.executionStore.CreateExecution(context.Background(), s.execution)
	s.NoError(err)

	// verify the execution was created
	readExecution, err := s.executionStore.GetExecution(context.Background(), s.execution.ID)
	s.NoError(err)
	s.EqualValues(s.execution, readExecution)

	// verify a history entry was created
	history, err := s.executionStore.GetExecutionHistory(context.Background(), s.execution.ID)
	s.NoError(err)
	s.Len(history, 1)
	s.verifyHistory(history[0], readExecution, store.ExecutionStateUndefined, newExecutionComment)
}

func (s *Suite) TestCreateExecution_AlreadyExists() {
	err := s.executionStore.CreateExecution(context.Background(), s.execution)
	s.NoError(err)

	err = s.executionStore.CreateExecution(context.Background(), s.execution)
	s.Error(err)
}

func (s *Suite) TestCreateExecution_InvalidState() {
	s.execution.State = store.ExecutionStateBidAccepted
	err := s.executionStore.CreateExecution(context.Background(), s.execution)
	s.Error(err)
}

func (s *Suite) TestGetExecution_DoesntExist() {
	_, err := s.executionStore.GetExecution(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
}

func (s *Suite) TestGetExecutions() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	readExecutions, err := s.executionStore.GetExecutions(ctx, s.execution.Job.ID())
	s.NoError(err)
	s.Len(readExecutions, 1)
	s.Equal(s.execution, readExecutions[0])

	// Create another execution for the same job
	anotherExecution := newExecution()
	anotherExecution.Job = s.execution.Job

	err = s.executionStore.CreateExecution(ctx, anotherExecution)
	s.NoError(err)

	readExecutions, err = s.executionStore.GetExecutions(ctx, s.execution.Job.ID())
	s.NoError(err)
	s.Len(readExecutions, 2)
	s.EqualValues(s.execution, readExecutions[0])

	anotherExecution.CreateTime = anotherExecution.CreateTime.Round(0)
	anotherExecution.UpdateTime = anotherExecution.UpdateTime.Round(0)
	readExecutions[1].UpdateTime = readExecutions[1].UpdateTime.Round(0)
	s.EqualValues(anotherExecution, readExecutions[1])
}

func (s *Suite) TestGetExecutions_DoesntExist() {
	_, err := s.executionStore.GetExecutions(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestUpdateExecution() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
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
	s.Equal(s.execution.Version+1, readExecution.Version)

	// verify a new history entry was created
	history, err := s.executionStore.GetExecutionHistory(ctx, s.execution.ID)
	s.NoError(err)
	s.Len(history, 2)
	s.verifyHistory(history[1], readExecution, s.execution.State, request.Comment)
}

func (s *Suite) TestUpdateExecution_ConditionsPass() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedState:   s.execution.State,
		ExpectedVersion: s.execution.Version,
		NewState:        store.ExecutionStatePublishing,
		Comment:         "Hello There!",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.NoError(err)

	// verify the update happened as expected
	readExecution, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	s.NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.execution.Version+1, readExecution.Version)
}

func (s *Suite) TestUpdateExecution_ConditionsStateFail() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
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
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedVersion: s.execution.Version + 99,
		NewState:        store.ExecutionStatePublishing,
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.ErrorAs(err, &store.ErrInvalidExecutionVersion{})
}

func (s *Suite) TestDeleteExecution() {
	err := s.executionStore.CreateExecution(context.Background(), s.execution)
	s.NoError(err)

	err = s.executionStore.DeleteExecution(context.Background(), s.execution.ID)
	s.NoError(err)

	_, err = s.executionStore.GetExecution(context.Background(), s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})

	_, err = s.executionStore.GetExecutions(context.Background(), s.execution.Job.ID())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestDeleteExecution_MultiEntries() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	// second execution with same jobID
	secondExecution := newExecution()
	secondExecution.Job = s.execution.Job
	_ = s.executionStore.CreateExecution(ctx, secondExecution)

	// third execution with different jobID
	thirdExecution := newExecution()
	err = s.executionStore.CreateExecution(ctx, thirdExecution)
	s.NoError(err)

	// validate pre-state
	firstJobExecutions, err := s.executionStore.GetExecutions(ctx, s.execution.Job.ID())
	s.NoError(err)
	s.Len(firstJobExecutions, 2)

	secondJobExecutions, err := s.executionStore.GetExecutions(ctx, thirdExecution.Job.ID())
	s.NoError(err)
	s.Len(secondJobExecutions, 1)

	// delete first execution
	err = s.executionStore.DeleteExecution(ctx, s.execution.ID)
	s.NoError(err)
	_, err = s.executionStore.GetExecution(ctx, s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	executions, err := s.executionStore.GetExecutions(ctx, s.execution.Job.ID())
	s.NoError(err)
	s.Len(executions, 1)

	// delete second execution
	err = s.executionStore.DeleteExecution(ctx, secondExecution.ID)
	s.NoError(err)
	_, err = s.executionStore.GetExecution(ctx, secondExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	executions, err = s.executionStore.GetExecutions(ctx, secondExecution.Job.ID())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})

	// delete third execution
	err = s.executionStore.DeleteExecution(ctx, thirdExecution.ID)
	s.NoError(err)
	_, err = s.executionStore.GetExecution(ctx, thirdExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	_, err = s.executionStore.GetExecutions(ctx, thirdExecution.Job.ID())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *Suite) TestDeleteExecution_DoesntExist() {
	err := s.executionStore.DeleteExecution(context.Background(), uuid.NewString())
	s.Error(err)
	s.Contains(err.Error(), "execution not found")
}

func (s *Suite) TestGetExecutionHistory_DoesntExist() {
	_, err := s.executionStore.GetExecutionHistory(context.Background(), uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionHistoryNotFound{})
}

func newExecution() store.Execution {
	return *store.NewExecution(
		uuid.NewString(),
		model.Job{
			Metadata: model.Metadata{
				ID: uuid.NewString(),
			},
		},
		"nodeID-1",
		model.ResourceUsageData{
			CPU:    1,
			Memory: 2,
		})
}

func (s *Suite) verifyHistory(history store.ExecutionHistory, newExecution store.Execution, previousState store.ExecutionState, comment string) {
	s.Equal(previousState, history.PreviousState)
	s.Equal(newExecution.ID, history.ExecutionID)
	s.Equal(newExecution.State, history.NewState)
	s.Equal(newExecution.Version, history.NewVersion)
	s.Equal(newExecution.UpdateTime, history.Time)
	s.Equal(comment, history.Comment)
}
