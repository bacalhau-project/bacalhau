package inmemory

import (
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

	err = store.AddJob(executor.Job{
		ID:    jobId,
		State: map[string]executor.JobState{},
	})
	require.NoError(t, err)

	err = store.AddEvent(jobId, executor.JobEvent{
		JobID:     jobId,
		NodeID:    nodeId,
		EventName: executor.JobEventBid,
	})
	require.NoError(t, err)

	err = store.UpdateJobState(jobId, nodeId, executor.JobState{
		State: executor.JobStateBidding,
	})
	require.NoError(t, err)

	err = store.UpdateLocalMetadata(jobId, executor.JobLocalMetadata{
		ComputeNodeSelected: true,
	})
	require.NoError(t, err)

	job, err := store.GetJob(jobId)
	require.NoError(t, err)
	require.Equal(t, jobId, job.ID)
	require.Equal(t, 1, len(job.Events))
	require.Equal(t, executor.JobEventBid, job.Events[0].EventName)
	require.Equal(t, 1, len(job.Job.State))
	require.Equal(t, executor.JobStateBidding, job.Job.State[nodeId].State)
	require.Equal(t, true, job.LocalMetadata.ComputeNodeSelected)

}
