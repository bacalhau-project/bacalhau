package sensors

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/compute/backend"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type RunningInfoProviderParams struct {
	Name          string
	BackendBuffer *backend.ServiceBuffer
}

type RunningInfoProvider struct {
	name          string
	backendBuffer *backend.ServiceBuffer
}

func NewRunningInfoProvider(params RunningInfoProviderParams) *RunningInfoProvider {
	return &RunningInfoProvider{
		name:          params.Name,
		backendBuffer: params.BackendBuffer,
	}
}

func (r RunningInfoProvider) GetDebugInfo() (model.DebugInfo, error) {
	executions := r.backendBuffer.RunningExecutions()
	summaries := make([]store.ExecutionSummary, 0, len(executions))
	for _, execution := range executions {
		summaries = append(summaries, store.NewExecutionSummary(execution))
	}
	bytes, err := model.JSONMarshalWithMax(summaries)
	if err != nil {
		return model.DebugInfo{}, fmt.Errorf("failed to marshal execution summaries: %w", err)
	}

	return model.DebugInfo{
		Component: r.name,
		Info:      string(bytes),
	}, nil
}

// compile-time check that we implement the interface
var _ model.DebugInfoProvider = (*RunningInfoProvider)(nil)
