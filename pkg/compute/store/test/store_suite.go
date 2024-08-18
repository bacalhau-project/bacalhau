//go:build unit || !integration

package test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type StoreCreator func(ctx context.Context, dbpath string) (store.ExecutionStore, error)

type StoreSuite struct {
	suite.Suite
	ctx                 context.Context
	executionStore      store.ExecutionStore
	localExecutionState store.LocalExecutionState
	execution           *models.Execution
	storeCreator        StoreCreator
	dbPath              string
}

func (s *StoreSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
}

func (s *StoreSuite) SetupTest() {
	var err error
	s.ctx = context.Background()
	s.dbPath = s.T().TempDir()
	s.executionStore, err = s.storeCreator(s.ctx, s.dbPath)
	require.NoError(s.T(), err)
	s.execution = mock.ExecutionForJob(mock.Job())
	s.localExecutionState = *store.NewLocalExecutionState(s.execution)
}

func (s *StoreSuite) TearDownTest() {
	if s.executionStore != nil {
		s.NoError(s.executionStore.Close(s.ctx))
	}
	_ = os.Remove(s.dbPath)
}

func RunStoreSuite(t *testing.T, creator StoreCreator) {
	s := new(StoreSuite)
	s.storeCreator = creator
	suite.Run(t, s)
}

func (s *StoreSuite) TestCreateExecution() {
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
	s.verifyHistory(history[0], readExecution, store.ExecutionStateUndefined, store.NewExecutionMessage)
}

func (s *StoreSuite) TestCreateExecutionAlreadyExists() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	err = s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Error(err)
}

func (s *StoreSuite) TestCreateExecutionInvalidState() {
	s.localExecutionState.State = store.ExecutionStateRunning
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Error(err)
}

func (s *StoreSuite) TestGetExecutionDoesntExist() {
	_, err := s.executionStore.GetExecution(s.ctx, uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
}

func (s *StoreSuite) TestGetExecutions() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	readExecutions, err := s.executionStore.GetExecutions(s.ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(readExecutions, 1)
	s.Equal(s.localExecutionState, readExecutions[0])

	// Create another execution for the same job
	anotherExecution := mock.ExecutionForJob(s.execution.Job)
	anotherExecutionState := *store.NewLocalExecutionState(anotherExecution)
	err = s.executionStore.CreateExecution(s.ctx, anotherExecutionState)
	s.Require().NoError(err)

	readExecutions, err = s.executionStore.GetExecutions(s.ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(readExecutions, 2)
	s.Equal(s.localExecutionState, readExecutions[0])
	s.Equal(anotherExecutionState, readExecutions[1])
}

func (s *StoreSuite) TestGetLiveExecutions() {
	localExec := store.NewLocalExecutionState(s.execution)
	err := s.executionStore.CreateExecution(s.ctx, *localExec)
	s.Require().NoError(err)

	err = s.executionStore.UpdateExecutionState(s.ctx, store.UpdateExecutionStateRequest{
		ExecutionID: s.execution.ID,
		NewState:    store.ExecutionStateRunning,
	})
	s.Require().NoError(err)

	execs, err := s.executionStore.GetLiveExecutions(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, len(execs))
	s.Require().Equal(s.execution.ID, execs[0].Execution.ID)
}

func (s *StoreSuite) TestFullLiveExecutions() {
	requireLive := func(executionID string) {
		execs, err := s.executionStore.GetLiveExecutions(s.ctx)
		s.Require().NoError(err)
		s.Require().Equal(1, len(execs))
		s.Require().Equal(executionID, execs[0].Execution.ID)
	}
	requireNotLive := func(executionID string) {
		execs, err := s.executionStore.GetLiveExecutions(s.ctx)
		s.Require().NoError(err)
		s.Require().Equal([]store.LocalExecutionState{}, execs)
	}

	localExec := store.NewLocalExecutionState(s.execution)
	err := s.executionStore.CreateExecution(s.ctx, *localExec)
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
			err = s.executionStore.UpdateExecutionState(s.ctx, store.UpdateExecutionStateRequest{
				ExecutionID: s.execution.ID,
				NewState:    tc.state,
			})
			s.Require().NoError(err)

			tc.requirement(s.execution.ID)
		})
	}
}

func (s *StoreSuite) TestGetMultipleLiveExecutions() {
	for i := 0; i < 3; i++ {
		exec := mock.ExecutionForJob(mock.Job())
		exec.ID = fmt.Sprintf("%d", i+1)
		localExec := store.NewLocalExecutionState(exec)
		err := s.executionStore.CreateExecution(s.ctx, *localExec)
		s.Require().NoError(err)

		err = s.executionStore.UpdateExecutionState(s.ctx, store.UpdateExecutionStateRequest{
			ExecutionID: exec.ID,
			NewState:    store.ExecutionStateRunning,
		})
		s.Require().NoError(err)
	}

	execs, err := s.executionStore.GetLiveExecutions(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(3, len(execs))

	// We want to make sure the executions are returned with increasing update times
	// that is, oldest first. On windows, the Update times are the same and so we
	// allow them to compare as <=.
	s.Require().LessOrEqual(execs[0].UpdateTime, execs[1].UpdateTime)
	s.Require().LessOrEqual(execs[1].UpdateTime, execs[2].UpdateTime)
}

func (s *StoreSuite) TestGetExecutionsDoesntExist() {
	_, err := s.executionStore.GetExecutions(s.ctx, uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *StoreSuite) TestUpdateExecution() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID: s.execution.ID,
		NewState:    store.ExecutionStatePublishing,
		Comment:     "Hello There!",
	}
	err = s.executionStore.UpdateExecutionState(s.ctx, request)
	s.Require().NoError(err)

	// verify the update happened as expected
	readExecution, err := s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.localExecutionState.Revision+1, readExecution.Revision)

	// verify a new history entry was created
	history, err := s.executionStore.GetExecutionHistory(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Len(history, 2)
	s.verifyHistory(history[1], readExecution, s.localExecutionState.State, request.Comment)
}

func (s *StoreSuite) TestUpdateExecutionConditionsPass() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:      s.execution.ID,
		ExpectedStates:   []store.LocalExecutionStateType{s.localExecutionState.State},
		ExpectedRevision: s.localExecutionState.Revision,
		NewState:         store.ExecutionStatePublishing,
		Comment:          "Hello There!",
	}
	err = s.executionStore.UpdateExecutionState(s.ctx, request)
	s.Require().NoError(err)

	// verify the update happened as expected
	readExecution, err := s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.localExecutionState.Revision+1, readExecution.Revision)
}

func (s *StoreSuite) TestGetExecutionCount() {
	states := []store.LocalExecutionStateType{
		store.ExecutionStateBidAccepted,
		store.ExecutionStateBidAccepted,
		store.ExecutionStateBidAccepted,
		store.ExecutionStateCompleted,
		store.ExecutionStateCompleted,
	}

	for _, state := range states {
		execution := mock.ExecutionForJob(mock.Job())
		executionState := *store.NewLocalExecutionState(execution)
		err := s.executionStore.CreateExecution(s.ctx, executionState)
		s.Require().NoError(err)

		request := store.UpdateExecutionStateRequest{
			ExecutionID:      execution.ID,
			ExpectedStates:   []store.LocalExecutionStateType{executionState.State},
			ExpectedRevision: executionState.Revision,
			NewState:         state,
			Comment:          "Hello There!",
		}

		err = s.executionStore.UpdateExecutionState(s.ctx, request)
		s.Require().NoError(err)
	}

	// Close and re-open the execution store so we can
	// re-populate the counter
	s.Require().NoError(s.executionStore.Close(s.ctx))
	var err error
	s.executionStore, err = s.storeCreator(s.ctx, s.dbPath)
	s.Require().NoError(err)

	c, err := s.executionStore.GetExecutionCount(s.ctx, store.ExecutionStateCompleted)
	s.Require().NoError(err)
	s.Equal(uint64(2), c)
}

func (s *StoreSuite) TestStateCounterChange() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	request := store.UpdateExecutionStateRequest{
		ExecutionID:      s.execution.ID,
		ExpectedStates:   []store.LocalExecutionStateType{s.localExecutionState.State},
		ExpectedRevision: s.localExecutionState.Revision,
		NewState:         store.ExecutionStateBidAccepted,
		Comment:          "Starting",
	}
	err = s.executionStore.UpdateExecutionState(s.ctx, request)
	s.Require().NoError(err)

	accepted, err := s.executionStore.GetExecutionCount(s.ctx, store.ExecutionStateBidAccepted)
	s.Require().NoError(err)
	completed, err := s.executionStore.GetExecutionCount(s.ctx, store.ExecutionStateCompleted)
	s.Require().NoError(err)
	s.Equal(uint64(1), accepted)
	s.Equal(uint64(0), completed)

	request = store.UpdateExecutionStateRequest{
		ExecutionID:      s.execution.ID,
		ExpectedStates:   []store.LocalExecutionStateType{store.ExecutionStateBidAccepted},
		ExpectedRevision: s.localExecutionState.Revision + 1,
		NewState:         store.ExecutionStateCompleted,
		Comment:          "Completed",
	}
	err = s.executionStore.UpdateExecutionState(s.ctx, request)
	s.Require().NoError(err)

	accepted, err = s.executionStore.GetExecutionCount(s.ctx, store.ExecutionStateBidAccepted)
	s.Require().NoError(err)
	completed, err = s.executionStore.GetExecutionCount(s.ctx, store.ExecutionStateCompleted)
	s.Require().NoError(err)
	s.Equal(uint64(0), accepted)
	s.Equal(uint64(1), completed)
}

func (s *StoreSuite) TestUpdateExecutionConditionsStateFail() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:    s.execution.ID,
		ExpectedStates: []store.LocalExecutionStateType{store.ExecutionStateBidAccepted},
		NewState:       store.ExecutionStatePublishing,
	}
	err = s.executionStore.UpdateExecutionState(s.ctx, request)
	s.ErrorAs(err, &store.ErrInvalidExecutionState{})
}

func (s *StoreSuite) TestUpdateExecutionConditionsRevisionFail() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	// update with no conditions
	request := store.UpdateExecutionStateRequest{
		ExecutionID:      s.execution.ID,
		ExpectedRevision: s.localExecutionState.Revision + 99,
		NewState:         store.ExecutionStatePublishing,
	}
	err = s.executionStore.UpdateExecutionState(s.ctx, request)
	s.ErrorAs(err, &store.ErrInvalidExecutionRevision{})
}

func (s *StoreSuite) TestDeleteExecution() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	err = s.executionStore.DeleteExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)

	_, err = s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})

	_, err = s.executionStore.GetExecutions(s.ctx, s.execution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *StoreSuite) TestDeleteExecutionMultiEntries() {
	err := s.executionStore.CreateExecution(s.ctx, s.localExecutionState)
	s.Require().NoError(err)

	// second execution with same jobID
	secondExecution := mock.ExecutionForJob(s.execution.Job)
	secondExecutionState := *store.NewLocalExecutionState(secondExecution)
	err = s.executionStore.CreateExecution(s.ctx, secondExecutionState)
	require.NoError(s.T(), err)

	// third execution with different jobID
	thirdExecution := mock.ExecutionForJob(mock.Job())
	thirdExecutionState := *store.NewLocalExecutionState(thirdExecution)
	err = s.executionStore.CreateExecution(s.ctx, thirdExecutionState)
	s.Require().NoError(err)

	// validate pre-state
	firstJobExecutions, err := s.executionStore.GetExecutions(s.ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(firstJobExecutions, 2)

	secondJobExecutions, err := s.executionStore.GetExecutions(s.ctx, thirdExecution.JobID)
	s.Require().NoError(err)
	s.Len(secondJobExecutions, 1)
	// delete first execution
	err = s.executionStore.DeleteExecution(s.ctx, s.execution.ID)
	s.Require().NoError(err)
	_, err = s.executionStore.GetExecution(s.ctx, s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	executions, err := s.executionStore.GetExecutions(s.ctx, s.execution.JobID)
	s.Require().NoError(err)
	s.Len(executions, 1)

	// delete second execution
	err = s.executionStore.DeleteExecution(s.ctx, secondExecution.ID)
	s.Require().NoError(err)
	_, err = s.executionStore.GetExecution(s.ctx, secondExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	_, err = s.executionStore.GetExecutions(s.ctx, secondExecution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})

	// delete third execution
	err = s.executionStore.DeleteExecution(s.ctx, thirdExecution.ID)
	s.Require().NoError(err)
	_, err = s.executionStore.GetExecution(s.ctx, thirdExecution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
	_, err = s.executionStore.GetExecutions(s.ctx, thirdExecution.JobID)
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForJob{})
}

func (s *StoreSuite) TestDeleteExecutionDoesntExist() {
	err := s.executionStore.DeleteExecution(s.ctx, uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionNotFound{})
}

func (s *StoreSuite) TestGetExecutionHistoryDoesntExist() {
	_, err := s.executionStore.GetExecutionHistory(s.ctx, uuid.NewString())
	s.ErrorAs(err, &store.ErrExecutionHistoryNotFound{})
}

func (s *StoreSuite) verifyHistory(history store.LocalStateHistory,
	newExecution store.LocalExecutionState, previousState store.LocalExecutionStateType, comment string) {
	s.Equal(previousState, history.PreviousState)
	s.Equal(newExecution.Execution.ID, history.ExecutionID)
	s.Equal(newExecution.State, history.NewState)
	s.Equal(newExecution.Revision, history.NewRevision)
	s.Equal(newExecution.UpdateTime, history.Time)
	s.Equal(comment, history.Comment)
}
