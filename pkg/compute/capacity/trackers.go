package capacity

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type LocalTrackerParams struct {
	MaxCapacity models.Resources
}

// LocalTracker keeps track of the current resource usage of the local node in-memory.
type LocalTracker struct {
	maxCapacity  models.Resources
	usedCapacity models.Resources
	mu           sync.Mutex
}

func NewLocalTracker(params LocalTrackerParams) *LocalTracker {
	return &LocalTracker{
		maxCapacity: params.MaxCapacity,
	}
}

func (t *LocalTracker) IsWithinLimits(ctx context.Context, usage models.Resources) bool {
	return usage.LessThanEq(t.maxCapacity)
}

func (t *LocalTracker) AddIfHasCapacity(ctx context.Context, usage models.Resources) *models.Resources {
	t.mu.Lock()
	defer t.mu.Unlock()

	newUsedCapacity := t.usedCapacity.Add(usage)
	if !newUsedCapacity.LessThanEq(t.maxCapacity) {
		return nil
	}

	// Allocate any GPUs that have been asked for but not chosen
	unspecifiedGPUs := math.Max(usage.GPU-uint64(len(usage.GPUs)), 0)
	availableGPUs := t.maxCapacity.Sub(t.usedCapacity).GPUs
	if unspecifiedGPUs > uint64(len(availableGPUs)) {
		return nil
	}
	usage.GPUs = append(usage.GPUs, availableGPUs[:unspecifiedGPUs]...)

	t.usedCapacity = *t.usedCapacity.Add(usage)
	return &usage
}

func (t *LocalTracker) GetAvailableCapacity(ctx context.Context) models.Resources {
	t.mu.Lock()
	defer t.mu.Unlock()
	return *t.maxCapacity.Sub(t.usedCapacity)
}

func (t *LocalTracker) GetMaxCapacity(ctx context.Context) models.Resources {
	return t.maxCapacity
}

func (t *LocalTracker) Remove(ctx context.Context, usage models.Resources) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usedCapacity = *t.usedCapacity.Sub(usage)
}

// compile-time check that LocalTracker implements Tracker
var _ Tracker = (*LocalTracker)(nil)
