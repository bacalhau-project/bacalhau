package sensors

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type CapacityInfoProviderParams struct {
	Name            string
	CapacityTracker capacity.Tracker
}

type CapacityInfoProvider struct {
	name            string
	capacityTracker capacity.Tracker
}

func NewCapacityInfoProvider(params CapacityInfoProviderParams) *CapacityInfoProvider {
	return &CapacityInfoProvider{
		name:            params.Name,
		capacityTracker: params.CapacityTracker,
	}
}

func (r CapacityInfoProvider) GetDebugInfo() (model.DebugInfo, error) {
	availableCapacity := r.capacityTracker.AvailableCapacity(context.Background())
	return model.DebugInfo{
		Component: r.name,
		Info:      availableCapacity.String(),
	}, nil
}

// compile-time check that we implement the interface
var _ model.DebugInfoProvider = (*CapacityInfoProvider)(nil)
