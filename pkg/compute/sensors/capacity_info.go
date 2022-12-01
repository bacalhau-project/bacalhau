package sensors

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type CapacityDebugInfoProviderParams struct {
	Name            string
	CapacityTracker capacity.Tracker
}

// CapacityDebugInfoProvider is a debug info provider that returns the available capacity of the node.
type CapacityDebugInfoProvider struct {
	name            string
	capacityTracker capacity.Tracker
}

func NewCapacityDebugInfoProvider(params CapacityDebugInfoProviderParams) *CapacityDebugInfoProvider {
	return &CapacityDebugInfoProvider{
		name:            params.Name,
		capacityTracker: params.CapacityTracker,
	}
}

func (r CapacityDebugInfoProvider) GetDebugInfo() (model.DebugInfo, error) {
	availableCapacity := r.capacityTracker.AvailableCapacity(context.Background())
	return model.DebugInfo{
		Component: r.name,
		Info:      availableCapacity.String(),
	}, nil
}

// compile-time check that we implement the interface
var _ model.DebugInfoProvider = (*CapacityDebugInfoProvider)(nil)
