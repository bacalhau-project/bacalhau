package capacity

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Tracker keeps track of the current resource usage of the compute node.
// The regular flow is to call AddIfHasCapacity before starting a new execution to reserve capacity, and Remove after
// the execution is done to release the reserved capacity.
type Tracker interface {
	// IsWithinLimits returns true if the given resource usage is within the limits of the compute node.
	// Limits refer to the total capacity of the compute node, and not to the currently available capacity.
	IsWithinLimits(ctx context.Context, usage models.Resources) bool
	// AddIfHasCapacity atomically adds the given resource usage to the tracker
	// if the compute node has capacity for it, returning the resource usage
	// that was added including any allocations that were made, or nil if the usage could not be added.
	AddIfHasCapacity(ctx context.Context, usage models.Resources) *models.Resources
	// GetAvailableCapacity returns the available capacity of the compute node.
	GetAvailableCapacity(ctx context.Context) models.Resources
	// GetMaxCapacity returns the total capacity of the compute node.
	GetMaxCapacity(ctx context.Context) models.Resources
	// Remove removes the given resource usage from the tracker.
	Remove(ctx context.Context, usage models.Resources)
}

// UsageTracker keeps track of the current resource usage of the compute node.
// Useful when tracking jobs in the queue pending and haven't started yet.
type UsageTracker interface {
	// Add adds the given resource usage to the tracker.
	Add(ctx context.Context, usage models.Resources)
	// Remove removes the given resource usage from the tracker.
	Remove(ctx context.Context, usage models.Resources)
	// GetUsedCapacity returns the current resource usage of the tracker
	GetUsedCapacity(ctx context.Context) models.Resources
}

// UsageCalculator calculates the resource usage of a job.
// Can also be used to populate the resource usage of a job with default values if not defined
type UsageCalculator interface {
	Calculate(ctx context.Context, execution *models.Execution, parsedUsage models.Resources) (*models.Resources, error)
}

// Provider returns the available capacity of a compute node.
// Implementation can return local node capacity if operating with a single node, or capacity of a cluster if compute
// is backed by a fleet of nodes.
type Provider interface {
	// GetAvailableCapacity returns the resources that are available for use by this node.
	GetAvailableCapacity(ctx context.Context) (models.Resources, error)

	// A set of human-readable strings that explains what this subprovider can detect.
	ResourceTypes() []string
}
