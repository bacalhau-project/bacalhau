package sync

import (
	"context"
	"errors"
	"reflect"

	sync "github.com/testground/sync-service"
)

func (c *DefaultClient) publish(ctx context.Context, topic string, payload interface{}) (int64, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch, err := c.makeRequest(ctx, &sync.Request{
		PublishRequest: &sync.PublishRequest{
			Topic:   topic,
			Payload: payload,
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

	return int64(res.PublishResponse.Seq), nil
}

func (c *DefaultClient) subscribe(ctx context.Context, key string, deref bool, val *typeValidator, ch interface{}) (sub *Subscription, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := c.ctx.Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	// sendFn is a closure that sends an element into the supplied ch,
	// performing necessary pointer to value conversions if necessary.
	//
	// sendFn will block if the receiver is not consuming from the channel.
	// If the context is closed, the send will be aborted, and the closure will
	// return a false value.
	sendFn := func(v reflect.Value) (sent bool) {
		if deref {
			v = v.Elem()
		}
		cases := []reflect.SelectCase{
			{Dir: reflect.SelectSend, Chan: reflect.ValueOf(ch), Send: v},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
		}
		_, _, ctxFired := reflect.Select(cases)
		return !ctxFired
	}

	req := &sync.Request{
		SubscribeRequest: &sync.SubscribeRequest{
			Topic: key,
		},
	}

	resCh, err := c.makeRequest(ctx, req)
	if err != nil {
		cancel()
		return nil, err
	}

	sub = &Subscription{make(chan error, 1)}

	go func() {
		defer cancel()

		for {
			select {
			case <-c.ctx.Done():
				sub.doneCh <- nil
				close(sub.doneCh)
				return
			case <-ctx.Done():
				sub.doneCh <- nil
				close(sub.doneCh)
				return
			case res, ok := <-resCh:
				if !ok {
					// Channel closed.
					sub.doneCh <- nil
					close(sub.doneCh)
					return
				}
				if res.Error != "" {
					sub.doneCh <- errors.New(res.Error)
					close(sub.doneCh)
					return
				}

				val, err := val.decodePayload(res.SubscribeResponse)
				if err != nil {
					c.log.Debugw("XREAD response: failed to decode message", "key", key, "error", err, "id", req.ID)
					continue
				}

				c.log.Debugw("dispatching message to subscriber", "key", key, "id", req.ID)
				if sent := sendFn(val); !sent {
					// we could not send value because context fired.
					// skip all further messages on this stream, and queue for
					// removal.
					c.log.Debugw("context was closed when dispatching message to subscriber; rm subscription", "key", key, "id", req.ID)
					sub.doneCh <- nil
					close(sub.doneCh)
					return
				}
			}
		}
	}()

	return sub, err
}
