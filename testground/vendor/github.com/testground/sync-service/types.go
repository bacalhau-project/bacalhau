package sync

import (
	"context"
	"io"
)

// Service is the implementation of a sync service. This service must support synchronization
// actions such as pub-sub and barriers.
type Service interface {
	io.Closer
	Publish(ctx context.Context, topic string, payload interface{}) (seq int, err error)
	Subscribe(ctx context.Context, topic string) (*subscription, error)
	Barrier(ctx context.Context, state string, target int) error
	SignalEntry(ctx context.Context, state string) (after int, err error)
}

// PublishRequest represents a publish request.
type PublishRequest struct {
	Topic   string      `json:"topic"`
	Payload interface{} `json:"payload"`
}

// PublishResponse represents a publish response.
type PublishResponse struct {
	Seq int `json:"seq"`
}

// SubscribeRequest represents a subscribe request.
type SubscribeRequest struct {
	Topic string `json:"topic"`
}

// BarrierRequest represents a barrier response.
type BarrierRequest struct {
	State  string `json:"state"`
	Target int    `json:"target"`
}

// SignalEntryRequest represents a signal entry request.
type SignalEntryRequest struct {
	State string `json:"state"`
}

// SignalEntryResponse represents a signal entry response.
type SignalEntryResponse struct {
	Seq int `json:"seq"`
}

// Request represents a request from the test instance to the sync service.
// The request ID must be present and one of the requests must be non-nil.
// The ID will be used on further responses.
type Request struct {
	ID                 string              `json:"id"`
	IsCancel           bool                `json:"is_cancel"`
	PublishRequest     *PublishRequest     `json:"publish,omitempty"`
	SubscribeRequest   *SubscribeRequest   `json:"subscribe,omitempty"`
	BarrierRequest     *BarrierRequest     `json:"barrier,omitempty"`
	SignalEntryRequest *SignalEntryRequest `json:"signal_entry,omitempty"`
}

// Response represents a response from the sync service to a test instance.
// The response ID must be present and one of the response types of Error must
// be non-nil. The ID is the same as the request ID.
type Response struct {
	ID                  string               `json:"id"`
	Error               string               `json:"error"`
	PublishResponse     *PublishResponse     `json:"publish"`
	SubscribeResponse   string               `json:"subscribe"` // JSON encoded subscribe response.
	SignalEntryResponse *SignalEntryResponse `json:"signal_entry"`
}
