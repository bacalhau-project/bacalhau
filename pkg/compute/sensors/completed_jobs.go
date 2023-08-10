package sensors

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type CompletedJobProvider struct {
	ExecutionStore store.ExecutionStore
}

func NewCompletedJobs(e store.ExecutionStore) *CompletedJobProvider {
	return &CompletedJobProvider{ExecutionStore: e}
}

// GetDebugInfo implements models.DebugInfoProvider
func (c *CompletedJobProvider) GetDebugInfo(ctx context.Context) (models.DebugInfo, error) {
	jobcounts, err := c.ExecutionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	return models.DebugInfo{
		Component: "jobsCompleted",
		Info:      jobcounts,
	}, err
}

var _ models.DebugInfoProvider = (*CompletedJobProvider)(nil)

// add a method to LocalState store interface to return an execution count for a given state.
