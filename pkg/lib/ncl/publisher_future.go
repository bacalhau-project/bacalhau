package ncl

import (
	"context"
	"sync"

	"github.com/nats-io/nats.go"
)

// pubFuture implementation matching jetstream
type pubFuture struct {
	msg    *nats.Msg
	result *Result
	err    error
	done   chan struct{}
	once   sync.Once
}

func newPubFuture(msg *nats.Msg) *pubFuture {
	return &pubFuture{
		msg:  msg,
		done: make(chan struct{}, 1),
	}
}

func (f *pubFuture) Done() <-chan struct{} {
	return f.done
}

// Result returns the processing result
func (f *pubFuture) Result() *Result {
	return f.result
}

func (f *pubFuture) Err() error {
	return f.err
}

func (f *pubFuture) Msg() *nats.Msg {
	return f.msg
}

func (f *pubFuture) Wait(ctx context.Context) error {
	select {
	case <-f.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *pubFuture) setError(err error) {
	f.once.Do(func() {
		f.err = err
		close(f.done)
	})
}

func (f *pubFuture) setResult(result *Result) {
	f.once.Do(func() {
		f.result = result
		close(f.done)
	})
}

var _ PubFuture = &pubFuture{}
