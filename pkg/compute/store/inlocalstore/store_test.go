//go:build unit || !integration

package inlocalstore

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
	s.Equal(uint(0), count)

}

func (s *Suite) TestOneExecutionReturnsOneCount() {
	//check jobStore is initialised to zero
	count, err := s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint(0), count)
	//create execution
	defaultJob, err := model.NewJobWithSaneProductionDefaults()
	s.NoError(err)
	execution := store.NewExecution(
		"id",
		*defaultJob,
		"defaultRequestorNodeID",
		model.ResourceUsageData{},
	)
	err = s.proxy.CreateExecution(context.Background(), *execution)
	s.NoError(err)
	err = s.proxy.UpdateExecutionState(context.Background(), store.UpdateExecutionStateRequest{
		ExecutionID: execution.ID,
		NewState:    store.ExecutionStateCompleted})
	s.NoError(err)
	count, err = s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
	s.NoError(err)
	s.Equal(uint(1), count)
}

func (s *Suite) TestConsecutiveExecutionsReturnCorrectJobCount() {
	idList := []string{"id1", "id2", "id3"}
	defaultJob, err := model.NewJobWithSaneProductionDefaults()
	s.NoError(err)
	// 3 executions
	for index, id := range idList {
		count, err := s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
		s.NoError(err)
		s.Equal(uint(index), count)

		execution := store.NewExecution(
			id,
			*defaultJob,
			"defaultRequestorNodeID",
			model.ResourceUsageData{},
		)
		err = s.proxy.CreateExecution(context.Background(), *execution)
		s.NoError(err)
		err = s.proxy.UpdateExecutionState(context.Background(), store.UpdateExecutionStateRequest{
			ExecutionID: execution.ID,
			NewState:    store.ExecutionStateCompleted})
		s.NoError(err)
	}

}

func (s *Suite) TestOnlyCompletedJobsIncreaseCounter() {
	defaultJob, err := model.NewJobWithSaneProductionDefaults()
	s.NoError(err)
	execution := store.NewExecution(
		"defaultID",
		*defaultJob,
		"defaultRequestorNodeID",
		model.ResourceUsageData{},
	)
	err = s.proxy.CreateExecution(context.Background(), *execution)
	s.NoError(err)
	for executionState := 0; executionState < 7; executionState++ {
		err = s.proxy.UpdateExecutionState(context.Background(), store.UpdateExecutionStateRequest{
			ExecutionID: execution.ID,
			NewState:    store.ExecutionState(executionState)})
		s.NoError(err)
		count, err := s.proxy.GetExecutionCount(context.Background(), store.ExecutionStateCompleted)
		s.NoError(err)
		s.Equal(uint(0), count)
	}
}
