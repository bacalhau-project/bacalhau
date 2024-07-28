package ncl

import (
	"context"
	"reflect"
)

// MessageHandler interface for processing messages
type MessageHandler interface {
	ShouldProcess(ctx context.Context, message *Message) bool
	HandleMessage(ctx context.Context, message *Message) error
}

// MessageFilter interface for filtering messages
type MessageFilter interface {
	ShouldFilter(metadata *Metadata) bool
}

// Checkpointer interface for managing checkpoints
type Checkpointer interface {
	Checkpoint(message *Message) error
	GetLastCheckpoint() (int64, error)
}

type MessageSerDe interface {
	Serialize(*RawMessage) ([]byte, error)
	Deserialize([]byte, *RawMessage) error
}

type PayloadSerDe interface {
	SerializePayload(*Metadata, any) ([]byte, error)
	DeserializePayload(*Metadata, reflect.Type, []byte) (any, error)
}

// Publisher publishes messages to a NATS server
type Publisher interface {
	Publish(ctx context.Context, event any) error
	PublishWithMetadata(ctx context.Context, metadata *Metadata, event any) error
}

// Subscriber subscribes to messages from a NATS server
type Subscriber interface {
	Subscribe(subjects ...string) error
	Close(ctx context.Context) error
}

// MessageHandlerFunc is a function type that implements MessageHandler
type MessageHandlerFunc func(ctx context.Context, message *Message) error

func (f MessageHandlerFunc) ShouldProcess(ctx context.Context, message *Message) bool {
	return true // Always process for this simple implementation
}

func (f MessageHandlerFunc) HandleMessage(ctx context.Context, message *Message) error {
	return f(ctx, message)
}

// MessageFilterFunc is a function type that implements MessageFilter
type MessageFilterFunc func(metadata *Metadata) bool

func (f MessageFilterFunc) ShouldFilter(metadata *Metadata) bool {
	return f(metadata)
}
