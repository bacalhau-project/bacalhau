package capacity

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// LocalUsageTracker keeps track of the resources used regardless of the total capacity.
// It is useful when tracking jobs in the queue pending and haven't started yet.
type LocalUsageTracker struct {
	usedCapacity models.Resources
	mu           sync.Mutex
}

func NewLocalUsageTracker() *LocalUsageTracker {
	return &LocalUsageTracker{}
}

func (t *LocalUsageTracker) Add(ctx context.Context, usage models.Resources) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usedCapacity = *t.usedCapacity.Add(usage)
}

func (t *LocalUsageTracker) Remove(ctx context.Context, usage models.Resources) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usedCapacity = *t.usedCapacity.Sub(usage)
}

func (t *LocalUsageTracker) GetUsedCapacity(ctx context.Context) models.Resources {
	return t.usedCapacity
}

// compile-time check that LocalUsageTracker implements Tracker
var _ UsageTracker = (*LocalUsageTracker)(nil)
