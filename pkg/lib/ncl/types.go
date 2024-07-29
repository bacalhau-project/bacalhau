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

// RawMessageSerDe interface for serializing and deserializing raw messages
// to and from byte slices.
type RawMessageSerDe interface {
	Serialize(*RawMessage) ([]byte, error)
	Deserialize([]byte) (*RawMessage, error)
}

// MessageSerDe interface for serializing and deserializing messages
// to and from raw messages.
type MessageSerDe interface {
	Serialize(message *Message) (*RawMessage, error)
	Deserialize(rawMessage *RawMessage, payloadType reflect.Type) (*Message, error)
}

// Publisher publishes messages to a NATS server
type Publisher interface {
	Publish(ctx context.Context, message *Message) error
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
