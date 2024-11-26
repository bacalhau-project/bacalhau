package dispatcher

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
)

type messageHandler struct {
	creator   transport.MessageCreator
	publisher ncl.OrderedPublisher
	state     *dispatcherState
}

func newMessageHandler(creator transport.MessageCreator, publisher ncl.OrderedPublisher, state *dispatcherState) *messageHandler {
	return &messageHandler{
		creator:   creator,
		publisher: publisher,
		state:     state,
	}
}

// HandleEvent processes a single event from the watcher.
// It creates and publishes a message asynchronously if needed.
// The method returns quickly, with actual publishing handled asynchronously.
func (h *messageHandler) HandleEvent(ctx context.Context, event watcher.Event) error {
	message, err := h.creator.CreateMessage(event)
	if err != nil {
		return newPublishError(fmt.Errorf("create message: %w", err))
	}

	if message == nil {
		h.state.updateLastObserved(event.SeqNum)
		return nil
	}

	if err = h.enrichAndPublish(ctx, message, event); err != nil {
		return err
	}

	h.state.updateLastObserved(event.SeqNum)
	return nil
}

func (h *messageHandler) enrichAndPublish(ctx context.Context, message *envelope.Message, event watcher.Event) error {
	// Add metadata
	message.WithMetadataValue(ncl.KeyMessageID, generateMsgID(event))
	message.WithMetadataValue(KeySeqNum, fmt.Sprint(event.SeqNum))

	// Prepare request
	request := ncl.NewPublishRequest(message)
	if message.Metadata.Has(ncl.KeySubject) {
		request = request.WithSubject(message.Metadata.Get(ncl.KeySubject))
	}

	// Publish request
	future, err := h.publisher.PublishAsync(ctx, request)
	if err != nil {
		return newPublishError(err)
	}

	// Track pending
	h.state.pending.Add(&pendingMessage{
		eventSeqNum: event.SeqNum,
		publishTime: time.Now(),
		future:      future,
	})

	return nil
}
