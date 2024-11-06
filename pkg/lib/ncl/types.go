package ncl

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

// MessageHandler interface for processing messages
type MessageHandler interface {
	ShouldProcess(ctx context.Context, message *envelope.Message) bool
	HandleMessage(ctx context.Context, message *envelope.Message) error
}

// MessageFilter interface for filtering messages
type MessageFilter interface {
	ShouldFilter(metadata *envelope.Metadata) bool
}

// Checkpointer interface for managing checkpoints
type Checkpointer interface {
	Checkpoint(message *envelope.Message) error
	GetLastCheckpoint() (int64, error)
}

// PublishRequest encapsulates the parameters needed to publish a message.
// Only one of Subject or SubjectPrefix should be set, not both.
type PublishRequest struct {
	// Message is the payload to be published (required)
	Message *envelope.Message
	// Subject is the exact NATS subject to publish to
	Subject string
	// SubjectPrefix is used to construct the final subject by appending additional information
	SubjectPrefix string
}

// NewPublishRequest creates a new PublishRequest
func NewPublishRequest(message *envelope.Message) PublishRequest {
	return PublishRequest{
		Message: message,
	}
}

// WithSubject sets the subject for the PublishRequest
func (r PublishRequest) WithSubject(subject string) PublishRequest {
	r.Subject = subject
	return r
}

// WithSubjectPrefix sets the subject prefix for the PublishRequest
func (r PublishRequest) WithSubjectPrefix(prefix string) PublishRequest {
	r.SubjectPrefix = prefix
	return r
}

// Publisher publishes messages to a NATS server
type Publisher interface {
	Publish(ctx context.Context, request PublishRequest) error
}

// Subscriber subscribes to messages from a NATS server
type Subscriber interface {
	Subscribe(subjects ...string) error
	Close(ctx context.Context) error
}

// MessageHandlerFunc is a function type that implements MessageHandler
type MessageHandlerFunc func(ctx context.Context, message *envelope.Message) error

func (f MessageHandlerFunc) ShouldProcess(ctx context.Context, message *envelope.Message) bool {
	return true // Always process for this simple implementation
}

func (f MessageHandlerFunc) HandleMessage(ctx context.Context, message *envelope.Message) error {
	return f(ctx, message)
}

// MessageFilterFunc is a function type that implements MessageFilter
type MessageFilterFunc func(metadata *envelope.Metadata) bool

func (f MessageFilterFunc) ShouldFilter(metadata *envelope.Metadata) bool {
	return f(metadata)
}
