package compute

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	compute_system "github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type CapacityProvider interface {
	Capacity() models.Resources
	RunningTracker() capacity.Tracker
	QueuedTracker() capacity.UsageTracker
	Calculator() capacity.UsageCalculator
}

func NewCapacityProvider(
	ctx context.Context,
	path string,
	cfg v2.Capacity,
	storages storage.StorageProvider,
) (*ComputeCapacityProvider, error) {
	resources, err := setupCapacity(ctx, compute_system.NewPhysicalCapacityProvider(path), cfg)
	if err != nil {
		return nil, err
	}
	calculator := capacity.NewChainedUsageCalculator(capacity.ChainedUsageCalculatorParams{
		Calculators: []capacity.UsageCalculator{
			// TODO(forrest) this merges the job defaults with the specified defaults.
			// It appears the side affect of this method is that it returns whatever is greater, the defaults, or
			// the job resources for subsequent bidding decisions in the bidder.
			capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{
				// the default resource to assign to jobs missing a resource config is the capacity of the compute node.
				Defaults: *resources,
			}),
			disk.NewDiskUsageCalculator(disk.DiskUsageCalculatorParams{
				Storages: storages,
			}),
		},
	})
	enqueuedUsageTracker := capacity.NewLocalUsageTracker()
	runningCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: *resources,
	})

	return &ComputeCapacityProvider{
		capacity:               *resources,
		calculator:             calculator,
		runningCapacityTracker: runningCapacityTracker,
		enqueuedUsageTracker:   enqueuedUsageTracker,
	}, nil
}

type ComputeCapacityProvider struct {
	capacity               models.Resources
	calculator             capacity.UsageCalculator
	runningCapacityTracker capacity.Tracker
	enqueuedUsageTracker   capacity.UsageTracker
}

func (c *ComputeCapacityProvider) Capacity() models.Resources {
	return c.capacity
}

func (c *ComputeCapacityProvider) RunningTracker() capacity.Tracker {
	return c.runningCapacityTracker
}

func (c *ComputeCapacityProvider) QueuedTracker() capacity.UsageTracker {
	return c.enqueuedUsageTracker
}

func (c *ComputeCapacityProvider) Calculator() capacity.UsageCalculator {
	return c.calculator
}

// SetupCapacity determines the compute capacity based on system capabilities and user configuration.
// It handles three scenarios:
//
// 1. If both Total and Allocated capacities are set in the config:
//   - Logs a warning that Allocated capacity will be ignored.
//   - Proceeds with Total capacity.
//
// 2. If only Allocated capacity is set:
//   - Scales the system capacity by the Allocated percentage.
//   - Returns the scaled capacity.
//
// 3. If only Total capacity is set:
//   - Parses the configured Total capacity.
//   - Verifies it doesn't exceed system capacity.
//   - Returns the configured capacity if valid.
//
// The method scales the system capacity when using Allocated settings, but does not
// scale the configured Total capacity. It returns an error if the configured Total
// capacity exceeds the system capacity.
func setupCapacity(ctx context.Context, provider *compute_system.PhysicalCapacityProvider, cfg v2.Capacity) (*models.Resources, error) {
	// TODO(forrest) [question] do we want to scale the configured total capacity by the allocated capacity?
	// Should we consider the configuration invalid if a user provides both an allocated and total capacity?
	// The latter could happen when merging several configuration files, maybe log a warning and state
	// the allocated capacity will be ignored since a total capacity was specified?
	if !cfg.Total.IsZero() && !cfg.Allocated.IsZero() {
		log.Warn().Msg("both allocated and total capacity are configured, ignoring allocated capacity scaler")
	}

	// determine the physical capacity of the host running the compute node
	systemCapacity, err := provider.GetTotalCapacity(ctx)
	if err != nil {
		return nil, fmt.Errorf("calculating system capacity: %w", err)
	}

	// if total capacity isn't configured, scale the capacity based on allocation
	if cfg.Total.IsZero() {
		scaledCapacity, err := systemCapacity.Scale(cfg.Allocated)
		if err != nil {
			return nil, fmt.Errorf("scaling system capacity by allocated capacity: %w", err)
		}
		log.Info().Stringer("capacity", scaledCapacity).Msg("scaled compute capacity")
		return scaledCapacity, nil
	}
	// else the user specified a total capacity
	configuredCapacity, err := models.ParseResourceConfig(cfg.Total)
	if err != nil {
		return nil, fmt.Errorf("parsing total capacity configuration: %w", err)
	}
	// if the user has requested a total capacity greater than the system capacity fail?
	if !configuredCapacity.LessThanEq(systemCapacity) {
		return nil, fmt.Errorf("configured total capaity (%s) cannot be greater than system capacity (%s)",
			configuredCapacity, &systemCapacity)
	}
	return configuredCapacity, nil
}
