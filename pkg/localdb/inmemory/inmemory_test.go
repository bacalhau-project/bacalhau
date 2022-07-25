package inmemory

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestInMemoryDataStore(t *testing.T) {

	jobId := "123"
	nodeId := "456"

	store, err := NewInMemoryDatastore()
	require.NoError(t, err)

	err = store.AddJob(context.Background(), executor.Job{
		ID:    jobId,
		State: map[string]executor.JobState{},
	})
	require.NoError(t, err)

	err = store.AddEvent(context.Background(), jobId, executor.JobEvent{
		JobID:        jobId,
		SourceNodeID: nodeId,
		EventName:    executor.JobEventBid,
	})
	require.NoError(t, err)

	err = store.UpdateExecutionState(context.Background(), jobId, nodeId, executor.JobState{
		State: executor.JobStateBidding,
	})
	require.NoError(t, err)

	err = store.AddLocalEvent(context.Background(), jobId, executor.JobLocalEvent{
		EventName: executor.JobLocalEventSelected,
	})
	require.NoError(t, err)

	job, err := store.GetJob(context.Background(), jobId)
	require.NoError(t, err)
	require.Equal(t, jobId, job.ID)
	require.Equal(t, 1, len(job.State))
	require.Equal(t, executor.JobStateBidding, job.State[nodeId].State)

	events, err := store.GetJobEvents(context.Background(), jobId)
	require.NoError(t, err)
	require.Equal(t, 1, len(events))
	require.Equal(t, executor.JobEventBid, events[0].EventName)

	localEvents, err := store.GetJobLocalEvents(context.Background(), jobId)
	require.NoError(t, err)
	require.Equal(t, 1, len(localEvents))
	require.Equal(t, executor.JobLocalEventSelected, localEvents[0].EventName)

}
