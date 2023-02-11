//go:build unit || !integration

package inmemory

import (
	"context"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestInMemoryDataStore(t *testing.T) {

	jobId := "12345678" // short ID is 8 chars long
	nodeId := "456"
	shardIndex := 1

	store, err := NewInMemoryDatastore()
	require.NoError(t, err)

	err = store.AddJob(context.Background(), &model.Job{
		Metadata: model.Metadata{
			ID: jobId,
		},
	})
	require.NoError(t, err)

	err = store.AddEvent(context.Background(), jobId, model.JobEvent{
		JobID:        jobId,
		SourceNodeID: nodeId,
		EventName:    model.JobEventBid,
	})
	require.NoError(t, err)

	err = store.UpdateShardState(context.Background(),
		jobId,
		nodeId,
		shardIndex,
		model.JobShardState{
			NodeID:               nodeId,
			ShardIndex:           shardIndex,
			State:                model.JobStateBidding,
			Status:               "hello",
			VerificationProposal: []byte("apples"),
		},
	)
	require.NoError(t, err)

	err = store.AddLocalEvent(context.Background(), jobId, model.JobLocalEvent{
		EventName: model.JobLocalEventSelected,
	})
	require.NoError(t, err)

	job, err := store.GetJob(context.Background(), jobId)
	require.NoError(t, err)
	require.Equal(t, jobId, job.Metadata.ID)

	// events, err := store.GetJobEvents(context.Background(), jobId)
	// require.NoError(t, err)
	// require.Equal(t, 1, len(events))
	// require.Equal(t, model.JobEventBid, events[0].EventName)

	localEvents, err := store.GetJobLocalEvents(context.Background(), jobId)
	require.NoError(t, err)
	require.Equal(t, 1, len(localEvents))
	require.Equal(t, model.JobLocalEventSelected, localEvents[0].EventName)

	jobState, err := store.GetJobState(context.Background(), jobId)
	require.NoError(t, err)

	nodeState, ok := jobState.Nodes[nodeId]
	require.True(t, ok)

	shardState, ok := nodeState.Shards[shardIndex]
	require.True(t, ok)

	require.Equal(t, model.JobStateBidding, shardState.State)
	require.Equal(t, "hello", shardState.Status)
}
