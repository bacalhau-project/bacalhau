package inlocalstore

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

func TestZeroExecutionsReturnsZeroCount(t *testing.T) {
	ctx := context.Background()
	system.InitConfigForTesting(t)
	//use inmemorystore
	proxy := NewStoreProxy(inmemory.NewStore())
	count, err := proxy.GetExecutionCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint(0), count)

}

func TestOneExecutionReturnsOneCount(t *testing.T) {
	ctx := context.Background()
	id := "fakeid1234"
	system.InitConfigForTesting(t)
	proxy := NewStoreProxy(inmemory.NewStore())
	//check jobStore is initialised to zero
	count, err := proxy.GetExecutionCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint(0), count)
	//create execution
	defaultJob, err := model.NewJobWithSaneProductionDefaults()
	require.NoError(t, err)
	execution := store.NewExecution(
		id,
		*defaultJob,
		"defaultRequestorNodeID",
		model.ResourceUsageData{},
	)
	err = proxy.CreateExecution(ctx, *execution)
	require.NoError(t, err)
	err = proxy.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
		ExecutionID: execution.ID,
		NewState:    store.ExecutionStateCompleted})
	require.NoError(t, err)
	count, err = proxy.GetExecutionCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint(1), count)
}

func TestConsecutiveExecutionsReturnCorrectJobCount(t *testing.T) {
	ctx := context.Background()
	idList := []string{"id1", "id2", "id3"}
	system.InitConfigForTesting(t)
	proxy := NewStoreProxy(inmemory.NewStore())
	defaultJob, err := model.NewJobWithSaneProductionDefaults()
	require.NoError(t, err)
	// 3 executions
	for index, id := range idList {
		count, err := proxy.GetExecutionCount(ctx)
		require.NoError(t, err)
		require.Equal(t, uint(index), count)

		execution := store.NewExecution(
			id,
			*defaultJob,
			"defaultRequestorNodeID",
			model.ResourceUsageData{},
		)
		err = proxy.CreateExecution(ctx, *execution)
		require.NoError(t, err)
		err = proxy.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID: execution.ID,
			NewState:    store.ExecutionStateCompleted})
		require.NoError(t, err)
	}

}

func TestOnlyCompletedJobsIncreaseCounter(t *testing.T) {
	ctx := context.Background()
	system.InitConfigForTesting(t)
	proxy := NewStoreProxy(inmemory.NewStore())
	defaultJob, err := model.NewJobWithSaneProductionDefaults()
	require.NoError(t, err)
	execution := store.NewExecution(
		"defaultID",
		*defaultJob,
		"defaultRequestorNodeID",
		model.ResourceUsageData{},
	)
	err = proxy.CreateExecution(ctx, *execution)
	require.NoError(t, err)
	for executionState := 0; executionState < 7; executionState++ {
		err = proxy.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID: execution.ID,
			NewState:    store.ExecutionState(executionState)})
		require.NoError(t, err)
		count, err := proxy.GetExecutionCount(ctx)
		require.NoError(t, err)
		require.Equal(t, uint(0), count)
	}
}

func TestSecondProxy(t *testing.T) {
	ctx := context.Background()
	idList := []string{"id1", "id2", "id3"}
	system.InitConfigForTesting(t)
	proxy := NewStoreProxy(inmemory.NewStore())
	defaultJob, err := model.NewJobWithSaneProductionDefaults()
	require.NoError(t, err)
	// 3 executions
	for index, id := range idList {
		count, err := proxy.GetExecutionCount(ctx)
		require.NoError(t, err)
		require.Equal(t, uint(index), count)

		execution := store.NewExecution(
			id,
			*defaultJob,
			"defaultRequestorNodeID",
			model.ResourceUsageData{},
		)
		err = proxy.CreateExecution(ctx, *execution)
		require.NoError(t, err)
		err = proxy.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID: execution.ID,
			NewState:    store.ExecutionStateCompleted})
		require.NoError(t, err)
	}
	count, err := proxy.GetExecutionCount(ctx)
	require.NoError(t, err)
	secondProxy := NewStoreProxy(inmemory.NewStore())
	secondCount, err := secondProxy.GetExecutionCount(ctx)
	require.NoError(t, err)
	require.Equal(t, count, secondCount)
}
