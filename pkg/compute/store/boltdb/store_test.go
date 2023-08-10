//go:build unit || !integration

package boltdb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	ctx            context.Context
	executionStore store.ExecutionStore
	execution      store.Execution
	dbFile         string
}

func (s *Suite) SetupTest() {
	s.ctx = context.Background()

	dir, _ := os.MkdirTemp("", "bacalhau-executionstore")
	s.dbFile = filepath.Join(dir, "test.boltdb")
	s.executionStore, _ = NewStore(s.ctx, s.dbFile)
	s.execution = newExecution()
}

func (s *Suite) TearDownTest() {
	os.Remove(s.dbFile)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestCreateExecution() {
	err := s.executionStore.CreateExecution(s.ctx, s.execution)
	s.NoError(err)

	// verify the execution was created
	readExecution, err := s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.NoError(err)
	s.Equal(s.execution, readExecution)

	// verify a history entry was created
	history, err := s.executionStore.GetExecutionHistory(s.ctx, s.execution.ID)
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
	s.execution.State = store.ExecutionStateRunning
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
	s.Equal(s.execution, readExecutions[0])
	s.Equal(anotherExecution, readExecutions[1])
}

func (s *Suite) TestGetLiveExecutions() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.Require().NoError(err)

	err = s.executionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: s.execution.ID,
		NewState:    store.ExecutionStateRunning,
	})
	s.Require().NoError(err)

	execs, err := s.executionStore.GetLiveExecutions(ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, len(execs))
	s.Require().Equal(s.execution.ID, execs[0].ID)
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

func (s *Suite) TestGetExecutionCount() {
	ctx := context.Background()

	states := []store.ExecutionState{
		store.ExecutionStateBidAccepted,
		store.ExecutionStateBidAccepted,
		store.ExecutionStateBidAccepted,
		store.ExecutionStateCompleted,
		store.ExecutionStateCompleted,
	}

	for _, state := range states {
		execution := newExecution()
		err := s.executionStore.CreateExecution(ctx, execution)
		s.NoError(err)

		request := store.UpdateExecutionStateRequest{
			ExecutionID:     execution.ID,
			ExpectedState:   execution.State,
			ExpectedVersion: execution.Version,
			NewState:        state,
			Comment:         "Hello There!",
		}

		err = s.executionStore.UpdateExecutionState(ctx, request)
		s.NoError(err)
	}

	// Close and re-open the execution store so we can
	// re-populate the counter
	s.executionStore.Close(s.ctx)
	s.executionStore, _ = NewStore(s.ctx, s.dbFile)

	c, err := s.executionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint64(2), c)
}

func (s *Suite) TestStateCounterChange() {
	ctx := context.Background()

	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	request := store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedState:   s.execution.State,
		ExpectedVersion: s.execution.Version,
		NewState:        store.ExecutionStateBidAccepted,
		Comment:         "Starting",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.NoError(err)

	accepted, err := s.executionStore.GetExecutionCount(ctx, store.ExecutionStateBidAccepted)
	s.NoError(err)
	completed, err := s.executionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint64(1), accepted)
	s.Equal(uint64(0), completed)

	request = store.UpdateExecutionStateRequest{
		ExecutionID:     s.execution.ID,
		ExpectedState:   store.ExecutionStateBidAccepted,
		ExpectedVersion: s.execution.Version + 1,
		NewState:        store.ExecutionStateCompleted,
		Comment:         "Completed",
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.NoError(err)

	accepted, err = s.executionStore.GetExecutionCount(ctx, store.ExecutionStateBidAccepted)
	s.NoError(err)
	completed, err = s.executionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint64(0), accepted)
	s.Equal(uint64(1), completed)

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
	ctx := s.ctx
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	// second execution with same jobID
	secondExecution := newExecution()
	secondExecution.Job = s.execution.Job
	err = s.executionStore.CreateExecution(ctx, secondExecution)
	require.NoError(s.T(), err)

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
	_, err = s.executionStore.GetExecutions(ctx, secondExecution.Job.ID())
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
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
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
