package sync

import (
	"context"
	"fmt"
)

type sugarOperations struct {
	Client
}

// PublishAndWait composes Publish and a Barrier. It first publishes the
// provided payload to the specified topic, then awaits for a barrier on the
// supplied state to reach the indicated target.
//
// If any operation fails, PublishAndWait short-circuits and returns a non-nil
// error and a negative sequence. If Publish succeeds, but the Barrier fails,
// the seq number will be greater than zero.
func (c *sugarOperations) PublishAndWait(ctx context.Context, topic *Topic, payload interface{}, state State, target int) (seq int64, err error) {
	seq, err = c.Publish(ctx, topic, payload)
	if err != nil {
		return -1, err
	}

	b, err := c.Barrier(ctx, state, target)
	if err != nil {
		return seq, err
	}

	<-b.C
	return seq, err
}

// MustPublishAndWait calls PublishAndWait, panicking if it errors.
//
// Suitable for shorthanding in test plans.
func (c *sugarOperations) MustPublishAndWait(ctx context.Context, topic *Topic, payload interface{}, state State, target int) (seq int64) {
	seq, err := c.PublishAndWait(ctx, topic, payload, state, target)
	if err != nil {
		panic(err)
	}
	return seq
}

// PublishSubscribe publishes the payload on the supplied Topic, then subscribes
// to it, sending paylods to the supplied channel.
//
// If any operation fails, PublishSubscribe short-circuits and returns a non-nil
// error and a negative sequence. If Publish succeeds, but Subscribe fails,
// the seq number will be greater than zero, but the returned Subscription will
// be nil, and the error, non-nil.
func (c *sugarOperations) PublishSubscribe(ctx context.Context, topic *Topic, payload interface{}, ch interface{}) (seq int64, sub *Subscription, err error) {
	seq, err = c.Publish(ctx, topic, payload)
	if err != nil {
		return -1, nil, err
	}
	sub, err = c.Subscribe(ctx, topic, ch)
	if err != nil {
		return seq, nil, err
	}
	return seq, sub, err
}

// MustPublishSubscribe calls PublishSubscribe, panicking if it errors.
//
// Suitable for shorthanding in test plans.
func (c *sugarOperations) MustPublishSubscribe(ctx context.Context, topic *Topic, payload interface{}, ch interface{}) (seq int64, sub *Subscription) {
	seq, sub, err := c.PublishSubscribe(ctx, topic, payload, ch)
	if err != nil {
		panic(err)
	}
	return seq, sub
}

// SignalAndWait composes SignalEntry and Barrier, signalling entry on the
// supplied state, and then awaiting until the required value has been reached.
//
// The returned error will be nil if the barrier was met successfully,
// or non-nil if the context expired, or some other error ocurred.
func (c *sugarOperations) SignalAndWait(ctx context.Context, state State, target int) (seq int64, err error) {
	//	rp := c.extractor(ctx)
	//	if rp == nil {
	//		return -1, ErrNoRunParameters
	//	}

	seq, err = c.SignalEntry(ctx, state)
	if err != nil {
		return -1, fmt.Errorf("failed while signalling entry to state %s: %w", state, err)
	}

	b, err := c.Barrier(ctx, state, target)
	if err != nil {
		return -1, fmt.Errorf("failed while setting barrier for state %s, with target %d: %w", state, target, err)
	}
	return seq, <-b.C
}

// MustSignalAndWait calls SignalAndWait, panicking if it errors.
//
// Suitable for shorthanding in test plans.
func (c *sugarOperations) MustSignalAndWait(ctx context.Context, state State, target int) (seq int64) {
	seq, err := c.SignalAndWait(ctx, state, target)
	if err != nil {
		panic(err)
	}
	return seq
}

// MustSignalEntry calls SignalEntry, panicking if it errors.
//
// Suitable for shorthanding in test plans.
func (c *sugarOperations) MustSignalEntry(ctx context.Context, state State) (current int64) {
	current, err := c.SignalEntry(ctx, state)
	if err != nil {
		panic(err)
	}
	return current
}

// MustBarrier calls Barrier, panicking if it errors.
//
// Suitable for shorthanding in test plans.
func (c *sugarOperations) MustBarrier(ctx context.Context, state State, required int) *Barrier {
	b, err := c.Barrier(ctx, state, required)
	if err != nil {
		panic(err)
	}
	return b
}

// MustPublish calls Publish, panicking if it errors.
//
// Suitable for shorthanding in test plans.
func (c *sugarOperations) MustPublish(ctx context.Context, topic *Topic, payload interface{}) (seq int64) {
	seq, err := c.Publish(ctx, topic, payload)
	if err != nil {
		panic(err)
	}
	return seq
}

// MustSubscribe calls Subscribe, panicking if it errors.
//
// Suitable for shorthanding in test plans.
func (c *sugarOperations) MustSubscribe(ctx context.Context, topic *Topic, ch interface{}) (sub *Subscription) {
	sub, err := c.Subscribe(ctx, topic, ch)
	if err != nil {
		panic(err)
	}
	return sub
}
