package sensors

import (
	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type RunningExecutionsInfoProviderParams struct {
	Name          string
	BackendBuffer *compute.ExecutorBuffer
}

// RunningExecutionsInfoProvider provides DebugInfo about the currently running executions.
// The info can be used for logging, metric, or to handle /debug API implementation.
type RunningExecutionsInfoProvider struct {
	name          string
	backendBuffer *compute.ExecutorBuffer
}

func NewRunningExecutionsInfoProvider(params RunningExecutionsInfoProviderParams) *RunningExecutionsInfoProvider {
	return &RunningExecutionsInfoProvider{
		name:          params.Name,
		backendBuffer: params.BackendBuffer,
	}
}

func (r RunningExecutionsInfoProvider) GetDebugInfo() (model.DebugInfo, error) {
	executions := r.backendBuffer.RunningExecutions()
	summaries := make([]store.ExecutionSummary, 0, len(executions))
	for _, execution := range executions {
		summaries = append(summaries, store.NewExecutionSummary(execution))
	}

	return model.DebugInfo{
		Component: r.name,
		Info:      summaries,
	}, nil
}

// compile-time check that we implement the interface
var _ model.DebugInfoProvider = (*RunningExecutionsInfoProvider)(nil)
