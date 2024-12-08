//go:generate mockgen --source types.go --destination mocks.go --package ncl
package ncl

import (
	"context"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

// MessageHandler interface for processing messages
type MessageHandler interface {
	ShouldProcess(ctx context.Context, message *envelope.Message) bool
	HandleMessage(ctx context.Context, message *envelope.Message) error
}

// RequestHandler processes incoming requests and returns responses
type RequestHandler interface {
	// HandleRequest processes a request message and returns a response
	HandleRequest(ctx context.Context, message *envelope.Message) (*envelope.Message, error)
}

// MessageFilter interface for filtering messages
type MessageFilter interface {
	ShouldFilter(metadata nats.Header) bool
}

// ProcessingNotifier provides callbacks for message processing events
type ProcessingNotifier interface {
	// OnProcessed is called when a message has been successfully processed
	OnProcessed(ctx context.Context, message *envelope.Message)
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

// Publisher interface combines publish and request operations
type Publisher interface {
	// Publish sends a message without expecting a response
	Publish(ctx context.Context, request PublishRequest) error

	// Request sends a message and waits for a response
	// Returns error if no response is received within the timeout
	Request(ctx context.Context, request PublishRequest) (*envelope.Message, error)
}

type OrderedPublisher interface {
	Publisher // Embed the Publisher interface
	PublishAsync(ctx context.Context, request PublishRequest) (PubFuture, error)
	Reset(ctx context.Context)
	Close(ctx context.Context) error
}

type PubFuture interface {
	// Done returns a receive only channel that can be used to wait for the future to be done.
	Done() <-chan struct{}

	// Err returns
	// If Done is not yet closed, Err returns nil.
	Err() error

	// Result returns the result of the future.
	// If Done is not yet closed, Result returns nil.
	Result() *Result

	// Msg returns the message that was sent to the server.
	Msg() *nats.Msg

	// Wait blocks until the future is done or the context is cancelled.
	Wait(ctx context.Context) error
}

// Subscriber subscribes to messages from a NATS server
type Subscriber interface {
	Subscribe(ctx context.Context, subjects ...string) error
	Close(ctx context.Context) error
}

// Responder handles incoming requests and sends back responses
type Responder interface {
	// Listen starts listening for requests of the given type
	Listen(ctx context.Context, messageType string, handler RequestHandler) error
	// Close stops listening for requests
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

// RequestHandlerFunc is a function type that implements RequestHandler
type RequestHandlerFunc func(ctx context.Context, message *envelope.Message) (*envelope.Message, error)

func (f RequestHandlerFunc) HandleRequest(ctx context.Context, message *envelope.Message) (*envelope.Message, error) {
	return f(ctx, message)
}

// MessageFilterFunc is a function type that implements MessageFilter
type MessageFilterFunc func(metadata nats.Header) bool

func (f MessageFilterFunc) ShouldFilter(metadata nats.Header) bool {
	return f(metadata)
}
