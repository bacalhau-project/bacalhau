package compute

import (
	"sync"

	"github.com/benbjohnson/clock"

	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

// HealthTracker monitors connection health and maintains status metrics.
// Thread-safe and uses an injectable clock for testing.
type HealthTracker struct {
	health nclprotocol.ConnectionHealth
	mu     sync.RWMutex
	clock  clock.Clock
}

// NewHealthTracker creates a new health tracker with the given clock
func NewHealthTracker(clock clock.Clock) *HealthTracker {
	return &HealthTracker{
		health: nclprotocol.ConnectionHealth{
			StartTime: clock.Now(),
		},
		clock: clock,
	}
}

// MarkConnected updates status when connection is established
func (ht *HealthTracker) MarkConnected() {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	ht.health.CurrentState = nclprotocol.Connected
	ht.health.ConnectedSince = ht.clock.Now()
	ht.health.LastSuccessfulHeartbeat = ht.clock.Now()
	ht.health.ConsecutiveFailures = 0
	ht.health.LastError = nil
	ht.health.CurrentState = nclprotocol.Connected
}

// MarkDisconnected updates status when connection is lost
func (ht *HealthTracker) MarkDisconnected(err error) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	ht.health.CurrentState = nclprotocol.Disconnected
	ht.health.LastError = err
	ht.health.ConsecutiveFailures++
}

// HeartbeatSuccess records successful heartbeat
func (ht *HealthTracker) HeartbeatSuccess() {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	ht.health.LastSuccessfulHeartbeat = ht.clock.Now()
}

// UpdateSuccess records successful node info update
func (ht *HealthTracker) UpdateSuccess() {
	ht.mu.Lock()
	defer ht.mu.Unlock()
	ht.health.LastSuccessfulUpdate = ht.clock.Now()
}

// GetState returns current connection state
func (ht *HealthTracker) GetState() nclprotocol.ConnectionState {
	ht.mu.RLock()
	defer ht.mu.RUnlock()
	return ht.health.CurrentState
}

// GetHealth returns a copy of current health status
func (ht *HealthTracker) GetHealth() nclprotocol.ConnectionHealth {
	ht.mu.RLock()
	defer ht.mu.RUnlock()
	return ht.health
}
