package sync

import (
	"context"
	"fmt"
	"reflect"
)

// Publish publishes an item on the supplied topic. The payload type must match
// the payload type on the Topic; otherwise Publish will error.
//
// This method returns synchronously, once the item has been published
// successfully, returning the sequence number of the new item in the ordered
// topic, or an error if one occurred, starting with 1 (for the first item).
//
// If error is non-nil, the sequence number must be disregarded.
func (c *DefaultClient) Publish(ctx context.Context, topic *Topic, payload interface{}) (int64, error) {
	rp := c.extractor(ctx)
	if rp == nil {
		return -1, ErrNoRunParameters
	}

	log := c.log.With("topic", topic.name)
	log.Debugw("publishing item on topic", "payload", payload)

	if !topic.validatePayload(payload) {
		err := fmt.Errorf("invalid payload type; expected: [*]%s, was: %T", topic.typ, payload)
		return -1, err
	}

	key := topic.Key(rp)
	log.Debugw("resolved key for publish", "key", key)

	seq, err := c.publish(ctx, key, payload)
	if err != nil {
		return -1, err
	}

	c.log.Debugw("successfully published item; sequence number obtained", "seq", seq)
	return seq, nil
}

// Subscribe subscribes to a topic, consuming ordered, typed elements from
// index 0, and sending them to channel ch.
//
// The supplied channel must be buffered, and its type must be a value or
// pointer type matching the topic type. If these conditions are unmet, this
// method will error immediately.
//
// The caller must consume from this channel promptly; failure to do so will
// backpressure the DefaultClient's subscription event loop.
func (c *DefaultClient) Subscribe(ctx context.Context, topic *Topic, ch interface{}) (sub *Subscription, err error) {
	rp := c.extractor(ctx)
	if rp == nil {
		return nil, ErrNoRunParameters
	}

	chv := reflect.ValueOf(ch)
	if k := chv.Kind(); k != reflect.Chan {
		return nil, fmt.Errorf("value is not a channel: %T", ch)
	}

	// compare the naked types; this makes the subscription work with pointer
	// or value channels.
	var deref bool
	chtyp := chv.Type().Elem()
	if chtyp.Kind() == reflect.Ptr {
		chtyp = chtyp.Elem()
	} else {
		deref = true
	}

	return c.subscribe(ctx, topic.Key(rp), deref, topic.typeValidator, ch)
}
