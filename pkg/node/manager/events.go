package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/benbjohnson/clock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// NodeEvent represents the type of event that can be emitted by the NodeEventEmitter.
type NodeEvent int

const (
	NodeEventApproved NodeEvent = iota
	NodeEventRejected
)

func (n NodeEvent) String() string {
	if n == NodeEventApproved {
		return "NodeEventApproved"
	} else if n == NodeEventRejected {
		return "NodeEventRejected"
	}
	return "UnknownNodeEvent"
}

// NodeEventCallback is a function that will be called when an event is emitted.
type NodeEventCallback func(info models.NodeInfo, event NodeEvent)

type NodeEventEmitterOption func(emitter *NodeEventEmitter)

// WithClock is an option that can be used to set the clock for the NodeEventEmitter. This is useful
// for testing purposes.
func WithClock(clock clock.Clock) NodeEventEmitterOption {
	return func(emitter *NodeEventEmitter) {
		emitter.clock = clock
	}
}

// NodeEventEmitter is a struct that will be used to emit events and register callbacks for those events.
// Events will be emitted by the node manager when a node is approved or rejected, and the expectation
// is that they will be consumed by the evaluation broker to create new evaluations.
// It is safe for concurrent use.
type NodeEventEmitter struct {
	mu        sync.Mutex
	callbacks map[NodeEvent][]NodeEventCallback
	clock     clock.Clock
}

func NewNodeEventEmitter(options ...NodeEventEmitterOption) *NodeEventEmitter {
	emitter := &NodeEventEmitter{
		callbacks: make(map[NodeEvent][]NodeEventCallback),
		clock:     clock.New(),
	}

	for _, option := range options {
		option(emitter)
	}

	return emitter
}

// RegisterCallback will register a callback for a specific event and add it to the list
// of existing callbacks, all of which will be called then that event is emitted.
func (e *NodeEventEmitter) RegisterCallback(event NodeEvent, callback NodeEventCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.callbacks[event] = append(e.callbacks[event], callback)
}

// EmitEvent will emit an event and call all the callbacks registered for that event. These callbacks
// are called in a goroutine and are expected to complete quickly.
func (e *NodeEventEmitter) EmitEvent(ctx context.Context, info models.NodeInfo, event NodeEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	completedChan := make(chan struct{})
	wg := sync.WaitGroup{}

	for _, callback := range e.callbacks[event] {
		wg.Add(1)
		go func(cb NodeEventCallback) {
			cb(info, event)
			wg.Done()
		}(callback)
	}

	// Wait for the waitgroup and then close the channel to signal completion. This allows
	// us to select on the completed channel as well as the timeout
	go func() {
		defer close(completedChan)
		wg.Wait()
	}()

	waitDuration := 1 * time.Second

	for {
		select {
		case <-completedChan:
			return nil
		case <-ctx.Done():
			return nil
		case <-e.clock.After(waitDuration):
			return fmt.Errorf("timed out waiting for %s event callbacks to complete", event.String())
		}
	}
}
