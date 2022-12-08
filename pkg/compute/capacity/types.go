package capacity

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// Tracker keeps track of the current resource usage of the compute node.
// The regular flow is to call AddIfHasCapacity before starting a new execution to reserve capacity, and Remove after
// the execution is done to release the reserved capacity.
type Tracker interface {
	// IsWithinLimits returns true if the given resource usage is within the limits of the compute node.
	// Limits refer to the total capacity of the compute node, and not to the currently available capacity.
	IsWithinLimits(ctx context.Context, usage model.ResourceUsageData) bool
	// AddIfHasCapacity atomically adds the given resource usage to the tracker if the compute node has capacity for it.
	AddIfHasCapacity(ctx context.Context, usage model.ResourceUsageData) bool
	// AvailableCapacity returns the available capacity of the compute node.
	AvailableCapacity(ctx context.Context) model.ResourceUsageData
	// Remove removes the given resource usage from the tracker.
	Remove(ctx context.Context, usage model.ResourceUsageData)
}

// UsageCalculator calculates the resource usage of a job.
// Can also be used to populate the resource usage of a job with default values if not defined
type UsageCalculator interface {
	Calculate(ctx context.Context, job model.Job, parsedUsage model.ResourceUsageData) (model.ResourceUsageData, error)
}

// Provider returns the available capacity of a compute node.
// Implementation can return local node capacity if operating with a single node, or capacity of a cluster if compute
// is backed by a fleet of nodes.
type Provider interface {
	AvailableCapacity(ctx context.Context) (model.ResourceUsageData, error)
}
