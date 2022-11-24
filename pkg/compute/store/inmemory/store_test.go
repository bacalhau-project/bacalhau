package inmemory

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	executionStore store.ExecutionStore
	execution      store.Execution
}

func (s *Suite) SetupTest() {
	s.executionStore = NewStore()
	s.execution = newExecution()
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestCreateExecution() {
	err := s.executionStore.CreateExecution(context.Background(), s.execution)
	s.NoError(err)

	readExecution, err := s.executionStore.GetExecution(context.Background(), s.execution.ID)
	s.NoError(err)
	s.Equal(s.execution, readExecution)
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
	execution, err := s.executionStore.GetExecution(context.Background(), uuid.NewString())
	s.NoError(err)
	s.Nil(execution)
}

func (s *Suite) TestGetExecution() {
	err := s.executionStore.CreateExecution(context.Background(), s.execution)
	s.NoError(err)

	readExecutions, err := s.executionStore.GetExecutions(context.Background(), s.execution.Shard.ID())
	s.NoError(err)
	s.Len(readExecutions, 1)
	s.Equal(s.execution, readExecutions[0])

	// Create another execution for the same shard
	anotherExecution := newExecution()
	anotherExecution.Shard = s.execution.Shard
	err = s.executionStore.CreateExecution(context.Background(), anotherExecution)
	s.NoError(err)

	readExecutions, err = s.executionStore.GetExecutions(context.Background(), s.execution.Shard.ID())
	s.NoError(err)
	s.Len(readExecutions, 2)
	s.Equal(s.execution, readExecutions[0])
	s.Equal(anotherExecution, readExecutions[1])
}

func (s *Suite) TestGetExecutions_DoesntExist() {
	executions, err := s.executionStore.GetExecutions(context.Background(), uuid.NewString())
	s.NoError(err)
	s.Empty(executions)
}

func (s *Suite) TestUpdateExecution() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	read, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	read.State = store.ExecutionStateBidAccepted
	s.NoError(err)
	s.Equal(s.execution, read)

	request := store.UpdateExecutionStateRequest{
		ExecutionID: s.execution.ID,
		NewState:    store.ExecutionStatePublishing,
	}
	err = s.executionStore.UpdateExecutionState(ctx, request)
	s.NoError(err)

	readExecution, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	s.NoError(err)
	s.Equal(request.NewState, readExecution.State)
	s.Equal(s.execution.Version+1, readExecution.Version)
}

func (s *Suite) TestDeleteExecution() {
	err := s.executionStore.CreateExecution(context.Background(), s.execution)
	s.NoError(err)

	err = s.executionStore.DeleteExecution(context.Background(), s.execution.ID)
	s.NoError(err)

	_, err = s.executionStore.GetExecution(context.Background(), s.execution.ID)
	s.ErrorAs(err, &store.ErrExecutionNotFound{})

	_, err = s.executionStore.GetExecutions(context.Background(), s.execution.Shard.ID())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForShard{})
}

func (s *Suite) TestDeleteExecution_MultiEntries() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	// second execution with same shardID
	secondExecution := newExecution()
	secondExecution.Shard = s.execution.Shard
	err = s.executionStore.CreateExecution(ctx, secondExecution)

	// third execution with different shardID
	thirdExecution := newExecution()
	err = s.executionStore.CreateExecution(ctx, thirdExecution)
	s.NoError(err)

	// validate pre-state
	firstShardExecutions, err := s.executionStore.GetExecutions(ctx, s.execution.Shard.ID())
	s.NoError(err)
	s.Len(firstShardExecutions, 2)

	secondShardExecutions, err := s.executionStore.GetExecutions(ctx, thirdExecution.Shard.ID())
	s.NoError(err)
	s.Len(secondShardExecutions, 1)

	// delete first execution
	err = s.executionStore.DeleteExecution(ctx, s.execution.ID)
	s.NoError(err)
	execution, err := s.executionStore.GetExecution(ctx, s.execution.ID)
	s.NoError(err)
	s.Nil(execution)
	executions, err := s.executionStore.GetExecutions(ctx, s.execution.Shard.ID())
	s.NoError(err)
	s.Len(executions, 1)

	// delete second execution
	err = s.executionStore.DeleteExecution(ctx, secondExecution.ID)
	s.NoError(err)
	execution, err = s.executionStore.GetExecution(ctx, secondExecution.ID)
	s.NoError(err)
	s.Nil(execution)
	executions, err = s.executionStore.GetExecutions(ctx, secondExecution.Shard.ID())
	s.NoError(err)
	s.Empty(executions)

	// delete third execution
	err = s.executionStore.DeleteExecution(ctx, thirdExecution.ID)
	s.NoError(err)
	execution, err = s.executionStore.GetExecution(ctx, thirdExecution.ID)
	s.NoError(err)
	s.Nil(execution)
	executions, err = s.executionStore.GetExecutions(ctx, thirdExecution.Shard.ID())
	s.NoError(err)
	s.Empty(executions)
}

func (s *Suite) TestDeleteExecution_DoesntExist() {
	err := s.executionStore.DeleteExecution(context.Background(), uuid.NewString())
	s.NoError(err)
}

func newExecution() store.Execution {
	return *store.NewExecution(
		uuid.NewString(),
		model.JobShard{
			Job:   &model.Job{ID: uuid.NewString()},
			Index: 1,
		},
		model.ResourceUsageData{
			CPU:    1,
			Memory: 2,
		})
}
