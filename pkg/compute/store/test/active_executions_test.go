package test

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/compute/store/inmemory"
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
	s.executionStore = inmemory.NewStore()
	s.execution = newExecution()
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestGetActiveExecution_Single() {
	ctx := context.Background()
	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	active, err := store.GetActiveExecution(ctx, s.executionStore, s.execution.Shard.ID())
	s.NoError(err)
	s.Equal(s.execution, active)
}

func (s *Suite) TestGetActiveExecution_Multiple() {
	ctx := context.Background()

	// create a newer execution with same shard as the previous one
	newerExecution := s.execution
	newerExecution.ID = uuid.NewString()
	newerExecution.Shard = s.execution.Shard
	newerExecution.UpdateTime = s.execution.UpdateTime.Add(1)

	err := s.executionStore.CreateExecution(ctx, s.execution)
	s.NoError(err)

	err = s.executionStore.CreateExecution(ctx, newerExecution)
	s.NoError(err)

	active, err := store.GetActiveExecution(ctx, s.executionStore, s.execution.Shard.ID())
	s.NoError(err)
	s.Equal(newerExecution, active)
}

func (s *Suite) TestGetActiveExecution_DoestExist() {
	_, err := store.GetActiveExecution(context.Background(), s.executionStore, s.execution.Shard.ID())
	s.ErrorAs(err, &store.ErrExecutionsNotFoundForShard{})
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
