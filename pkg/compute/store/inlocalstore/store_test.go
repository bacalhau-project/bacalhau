//go:build unit || !integration

package inlocalstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	proxy store.ExecutionStore
}

func (s *Suite) SetupTest() {
	var err error

	ctx := context.Background()
	f := filepath.Join(s.T().TempDir(), "test.db")
	store, err := boltdb.NewStore(ctx, f)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		store.Close(ctx)
	})

	s.proxy, err = NewPersistentExecutionStore(PersistentJobStoreParams{
		Store:   store,
		RootDir: s.T().TempDir(),
	})
	s.Require().NoError(err)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestZeroExecutionsReturnsZeroCount() {
	count, err := s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint64(0), count)

}

func (s *Suite) TestOneExecutionReturnsOneCount() {
	//check jobStore is initialised to zero
	count, err := s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint64(0), count)
	//create execution
	execution := mock.ExecutionForJob(mock.Job())
	executionState := *store.NewLocalExecutionState(execution, "nodeID")
	err = s.proxy.CreateExecution(context.Background(), executionState)
	s.NoError(err)
	err = s.proxy.UpdateExecutionState(context.Background(), store.UpdateExecutionStateRequest{
		ExecutionID: execution.ID,
		NewState:    store.ExecutionStateCompleted})
	s.NoError(err)
	count, err = s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint64(1), count)
}

func (s *Suite) TestConsecutiveExecutionsReturnCorrectJobCount() {
	job := mock.Job()
	// 3 executions
	for index := 0; index < 3; index++ {
		count, err := s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
		s.NoError(err)
		s.Equal(uint64(index), count)

		execution := mock.ExecutionForJob(job)
		executionState := *store.NewLocalExecutionState(execution, "nodeID")
		err = s.proxy.CreateExecution(context.Background(), executionState)
		s.NoError(err)
		err = s.proxy.UpdateExecutionState(context.Background(), store.UpdateExecutionStateRequest{
			ExecutionID: execution.ID,
			NewState:    store.ExecutionStateCompleted})
		s.NoError(err)
	}

}

func (s *Suite) TestOnlyCompletedJobsIncreaseCounter() {
	for _, executionStateType := range store.ExecutionStateTypes() {
		if executionStateType == store.ExecutionStateCompleted {
			continue
		}
		s.Run(executionStateType.String(), func() {
			execution := mock.ExecutionForJob(mock.Job())
			executionState := *store.NewLocalExecutionState(execution, "nodeID")
			err := s.proxy.CreateExecution(context.Background(), executionState)
			s.NoError(err)
			err = s.proxy.UpdateExecutionState(context.Background(), store.UpdateExecutionStateRequest{
				ExecutionID: execution.ID,
				NewState:    executionStateType})
			s.NoError(err)
			count, err := s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
			s.NoError(err)
			s.Equal(uint64(0), count)
		})
	}
}
