package distributed

import (
	"context"
	"encoding/json"
)

type Watcher[T any] struct {
	store    *DistributedObjectStore
	callback WatchCallbackFunc[T]
}

// A callback function that will be called with any new T instances
// and uses the returned boolean to determine whether to carry on.
type WatchCallbackFunc[T any] func(object T) bool

func NewWatcher[T any](store *DistributedObjectStore, callback WatchCallbackFunc[T]) *Watcher[T] {
	return &Watcher[T]{
		store:    store,
		callback: callback,
	}
}

func (w *Watcher[T]) Watch(ctx context.Context, prefix string) error {
	closeStream := make(chan struct{}, 1)
	readChannel, err := w.store.Stream(ctx, prefix, closeStream)
	if err != nil {
		return err
	}

	for {
		select {
		case data := <-readChannel:
			var target T
			_ = json.Unmarshal(data, &target)
			cont := w.callback(target)
			if !cont {
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}
