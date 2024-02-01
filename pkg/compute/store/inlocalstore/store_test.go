//go:build unit || !integration

package inlocalstore

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	proxy store.ExecutionStore
}

func (s *Suite) SetupTest() {
	var err error
	s.proxy, err = NewPersistentExecutionStore(PersistentJobStoreParams{
		Store:   inmemory.NewStore(),
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
