package capacity

import (
	"context"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
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

func (t *LocalTracker) AddIfHasCapacity(ctx context.Context, usage models.Resources) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	newUsedCapacity := t.usedCapacity.Add(usage)
	if newUsedCapacity.LessThanEq(t.maxCapacity) {
		t.usedCapacity = newUsedCapacity
		return true
	}
	return false
}

func (t *LocalTracker) GetAvailableCapacity(ctx context.Context) models.Resources {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.maxCapacity.Sub(t.usedCapacity)
}

func (t *LocalTracker) GetMaxCapacity(ctx context.Context) models.Resources {
	return t.maxCapacity
}

func (t *LocalTracker) Remove(ctx context.Context, usage models.Resources) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usedCapacity = t.usedCapacity.Sub(usage)
}

// compile-time check that LocalTracker implements Tracker
var _ Tracker = (*LocalTracker)(nil)
