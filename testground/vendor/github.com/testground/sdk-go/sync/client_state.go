package sync

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/testground/sdk-go/runtime"
	sync "github.com/testground/sync-service"
)

// Barrier sets a barrier on the supplied State that fires when it reaches its
// target value (or higher).
//
// The caller should monitor the channel C returned inside the Barrier object.
// It fires when the barrier reaches its target, is cancelled, or fails.
// If the barrier is satisfied, the value sent will be nil. C must not be closed
// by the caller.
//
// When the context fires, the context's error will be propagated instead. The
// same will occur if the DefaultClient's context fires.
//
// It is safe to use a non-cancellable context here, like the background
// context. No cancellation is needed unless you want to stop the process early.
func (c *DefaultClient) Barrier(ctx context.Context, state State, target int) (*Barrier, error) {
	// a barrier with target zero is satisfied immediately; log a warning as
	// this is probably programmer error.
	if target == 0 {
		c.log.Warnw("requested a barrier with target zero; satisfying immediately", "state", state)
		b := &Barrier{C: make(chan error, 1)}
		b.C <- nil
		close(b.C)
		return b, nil
	}

	rp := c.extractor(ctx)
	if rp == nil {
		return nil, ErrNoRunParameters
	}

	key := state.Key(rp)

	ctx, cancel := context.WithCancel(ctx)

	ch, err := c.makeRequest(ctx, &sync.Request{
		BarrierRequest: &sync.BarrierRequest{
			State:  key,
			Target: target,
		},
	})
	if err != nil {
		cancel()
		return nil, err
	}

	b := &Barrier{
		C: make(chan error, 1),
	}

	go func() {
		res, ok := <-ch
		if !ok {
			b.C <- errors.New("channel closed before getting response")
		} else if res.Error == "" {
			b.C <- nil
		} else {
			b.C <- errors.New(res.Error)
		}

		cancel()
	}()

	return b, nil
}

// SignalEntry increments the state counter by one, returning the value of the
// new value of the counter, or an error if the operation fails.
func (c *DefaultClient) SignalEntry(ctx context.Context, state State) (int64, error) {
	rp := c.extractor(ctx)
	if rp == nil {
		return -1, ErrNoRunParameters
	}

	key := state.Key(rp)

	c.log.Debugw("signalling entry to state", "key", key)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Increment a counter on the state key.
	ch, err := c.makeRequest(ctx, &sync.Request{
		SignalEntryRequest: &sync.SignalEntryRequest{
			State: key,
		},
	})
	if err != nil {
		return -1, err
	}

	res, ok := <-ch
	if !ok {
		return -1, errors.New("channel closed before getting response")
	}
	if res.Error != "" {
		return -1, errors.New(res.Error)
	}

	c.log.Debugw("new value of state", "key", key, "value", res.SignalEntryResponse.Seq)
	return int64(res.SignalEntryResponse.Seq), nil
}

// SignalEvent emits an event attached to a certain test plan.
func (c *DefaultClient) SignalEvent(ctx context.Context, event *runtime.Event) (err error) {
	rp := c.extractor(ctx)
	if rp == nil {
		return ErrNoRunParameters
	}

	key := fmt.Sprintf("run:%s:plan:%s:case:%s:run_events", rp.TestRun, rp.TestPlan, rp.TestCase)
	_, err = c.publish(ctx, key, event)
	return err
}

// SubscribeEvents monitors the events sent by a specific test plan. This function is used by Testground
// to monitor all emitted events by the testplans, in particular the terminal events, such as SuccessEvent,
// FailureEvent and CrashEvent.
func (c *DefaultClient) SubscribeEvents(ctx context.Context, rp *runtime.RunParams) (chan *runtime.Event, error) {
	ch := make(chan *runtime.Event)
	key := fmt.Sprintf("run:%s:plan:%s:case:%s:run_events", rp.TestRun, rp.TestPlan, rp.TestCase)

	_, err := c.subscribe(ctx, key, false, &typeValidator{reflect.TypeOf(&runtime.Event{})}, ch)
	if err != nil {
		return nil, err
	}

	return ch, nil
}
