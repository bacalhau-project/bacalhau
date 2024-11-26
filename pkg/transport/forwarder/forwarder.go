package forwarder

import (
	"context"
	"fmt"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
)

// Forwarder forwards events from a watcher to a destination in order.
// Unlike Dispatcher, it provides no delivery guarantees or recovery mechanisms.
type Forwarder struct {
	watcher   watcher.Watcher
	creator   transport.MessageCreator
	publisher ncl.OrderedPublisher

	running bool
	mu      sync.RWMutex
}

func New(
	publisher ncl.OrderedPublisher, watcher watcher.Watcher, creator transport.MessageCreator) (*Forwarder, error) {
	if publisher == nil {
		return nil, fmt.Errorf("publisher cannot be nil")
	}
	if watcher == nil {
		return nil, fmt.Errorf("watcher cannot be nil")
	}
	if creator == nil {
		return nil, fmt.Errorf("message creator cannot be nil")
	}

	f := &Forwarder{
		watcher:   watcher,
		creator:   creator,
		publisher: publisher,
	}

	if err := watcher.SetHandler(f); err != nil {
		return nil, fmt.Errorf("failed to set handler: %w", err)
	}

	return f, nil
}

func (f *Forwarder) Start(ctx context.Context) error {
	f.mu.Lock()
	if f.running {
		f.mu.Unlock()
		return fmt.Errorf("forwarder already running")
	}
	f.running = true
	f.mu.Unlock()

	return f.watcher.Start(ctx)
}

func (f *Forwarder) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.running {
		return nil
	}
	f.running = false
	f.watcher.Stop(ctx)
	return nil
}

func (f *Forwarder) HandleEvent(ctx context.Context, event watcher.Event) error {
	message, err := f.creator.CreateMessage(event)
	if err != nil {
		return fmt.Errorf("create message failed: %w", err)
	}
	if message == nil {
		return nil
	}

	// Add sequence number for ordering
	message.WithMetadataValue(transport.KeySeqNum, fmt.Sprint(event.SeqNum))
	message.WithMetadataValue(ncl.KeyMessageID, transport.GenerateMsgID(event))

	// Publish request
	request := ncl.NewPublishRequest(message)
	if message.Metadata.Has(ncl.KeySubject) {
		request = request.WithSubject(message.Metadata.Get(ncl.KeySubject))
	}

	if err = f.publisher.Publish(ctx, request); err != nil {
		return err
	}

	return nil
}
