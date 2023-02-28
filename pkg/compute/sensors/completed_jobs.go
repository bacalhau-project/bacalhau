package sensors

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type CompletedJobProvider struct {
	ExecutionStore store.ExecutionStore
}

func NewCompletedJobs(e store.ExecutionStore) *CompletedJobProvider {
	return &CompletedJobProvider{ExecutionStore: e}
}

// GetDebugInfo implements model.DebugInfoProvider
func (c *CompletedJobProvider) GetDebugInfo(ctx context.Context) (model.DebugInfo, error) {
	jobcounts := c.ExecutionStore.GetExecutionCount(ctx)
	return model.DebugInfo{
		Component: "jobsCompleted",
		Info:      jobcounts,
	}, nil
}

var _ model.DebugInfoProvider = (*CompletedJobProvider)(nil)

// add a method to Execution store interface to return an execution count for a given state.
