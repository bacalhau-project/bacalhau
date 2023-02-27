package sensors

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type CompletedJobs struct {
	ExecutionStore store.ExecutionStore
}

func NewCompletedJobs(e store.ExecutionStore) *CompletedJobs {
	return &CompletedJobs{ExecutionStore: e}
}

// GetDebugInfo implements model.DebugInfoProvider
func (c *CompletedJobs) GetDebugInfo(ctx context.Context) (model.DebugInfo, error) {
	jobcounts := c.ExecutionStore.GetExecutionCount(ctx)
	return model.DebugInfo{
		Component: "completed jobs:",
		Info:      jobcounts,
	}, nil
}

var _ model.DebugInfoProvider = (*CompletedJobs)(nil)

// add a method to Execution store interface to return an execution count for a given state.
