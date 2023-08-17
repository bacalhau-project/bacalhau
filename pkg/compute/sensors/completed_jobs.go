package sensors

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type CompletedJobProvider struct {
	ExecutionStore store.ExecutionStore
}

func NewCompletedJobs(e store.ExecutionStore) *CompletedJobProvider {
	return &CompletedJobProvider{ExecutionStore: e}
}

// GetDebugInfo implements model.DebugInfoProvider
func (c *CompletedJobProvider) GetDebugInfo(ctx context.Context) (model.DebugInfo, error) {
	jobcounts, err := c.ExecutionStore.GetExecutionCount(ctx, store.ExecutionStateCompleted)
	return model.DebugInfo{
		Component: "jobsCompleted",
		Info:      jobcounts,
	}, err
}

var _ model.DebugInfoProvider = (*CompletedJobProvider)(nil)

// add a method to LocalState store interface to return an execution count for a given state.
