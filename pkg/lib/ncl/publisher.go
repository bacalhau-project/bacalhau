package ncl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// publisher handles message publishing
type publisher struct {
	nc                   *nats.Conn
	messageSerializer    envelope.MessageSerializer
	messageSerDeRegistry *envelope.Registry
	name                 string
	destination          string
	destinationPrefix    string
}

// PublisherOption is a function type for configuring a publisher
type PublisherOption func(*publisher)

// WithPublisherMessageSerializer sets the message serializer for the publisher
func WithPublisherMessageSerializer(serializer envelope.MessageSerializer) PublisherOption {
	return func(p *publisher) {
		p.messageSerializer = serializer
	}
}

// WithPublisherMessageSerDeRegistry sets the payload registry for the publisher
func WithPublisherMessageSerDeRegistry(registry *envelope.Registry) PublisherOption {
	return func(p *publisher) {
		p.messageSerDeRegistry = registry
	}
}

// WithPublisherName sets the name for the publisher
func WithPublisherName(name string) PublisherOption {
	return func(p *publisher) {
		p.name = name
	}
}

// WithPublisherDestinationPrefix sets the destination prefix for the publisher
// The destination prefix is used to construct the subject for the message to be published
// The subject is constructed as follows: destinationPrefix + "." + messageType
// Caution: cannot be used with WithPublisherDestination
func WithPublisherDestinationPrefix(prefix string) PublisherOption {
	return func(p *publisher) {
		p.destinationPrefix = prefix
	}
}

// WithPublisherDestination sets the destination for the publisher
// The destination is used as the subject for the message to be published
// Caution: cannot be used with WithPublisherDestinationPrefix
func WithPublisherDestination(destination string) PublisherOption {
	return func(p *publisher) {
		p.destination = destination
	}
}

// defaultPublisher returns a publisher with default settings
func defaultPublisher(nc *nats.Conn) *publisher {
	return &publisher{
		nc:                nc,
		messageSerializer: envelope.NewSerializer(),
	}
}

// NewPublisher creates a new publisher with the given options
func NewPublisher(nc *nats.Conn, opts ...PublisherOption) (Publisher, error) {
	// Start with default publisher
	p := defaultPublisher(nc)

	// Apply all options
	for _, opt := range opts {
		opt(p)
	}

	// Validate the publisher
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("error validating publisher: %w", err)
	}

	return p, nil
}

// validate checks if the publisher is properly configured
func (p *publisher) validate() error {
	return errors.Join(
		validate.NotNil(p.nc, "NATS connection cannot be nil"),
		validate.NotBlank(p.name, "publisher name cannot be blank"),
		validate.NotNil(p.messageSerializer, "message serializer cannot be nil"),
		validate.NotNil(p.messageSerDeRegistry, "payload registry cannot be nil"),
	)
}

// Publish publishes a message to the NATS server
func (p *publisher) Publish(ctx context.Context, request PublishRequest) error {
	if err := p.validateRequest(request); err != nil {
		return err
	}

	p.enrichMetadata(request.Message)
	rMsg, err := p.messageSerDeRegistry.Serialize(request.Message)
	if err != nil {
		return fmt.Errorf("failed to serialize into raw message: %w", err)
	}

	subject := p.getSubject(request)

	data, err := p.messageSerializer.Serialize(rMsg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	err = p.nc.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// enrichMetadata adds metadata to the message, such as source and event time
func (p *publisher) enrichMetadata(message *envelope.Message) {
	message.Metadata.Set(KeySource, p.name)
	if !message.Metadata.Has(KeyEventTime) {
		message.Metadata.SetTime(KeyEventTime, time.Now())
	}
}

func (p *publisher) getSubject(request PublishRequest) string {
	if request.Subject != "" {
		return request.Subject
	}
	if request.SubjectPrefix != "" {
		return fmt.Sprintf("%s.%s", request.SubjectPrefix, request.Message.Metadata.Get(envelope.KeyMessageType))
	}
	if p.destination != "" {
		return p.destination
	}
	return fmt.Sprintf("%s.%s", p.destinationPrefix, request.Message.Metadata.Get(envelope.KeyMessageType))
}

func (p *publisher) validateRequest(request PublishRequest) error {
	err := errors.Join(
		validate.NotNil(request.Message, "cannot publish nil message"),
	)

	if request.Subject != "" && request.SubjectPrefix != "" {
		err = errors.Join(err, errors.New("cannot specify both subject and subject prefix"))
	}

	// if no p.destination or p.destinationPrefix is set, then the subject or subject prefix must be set
	if p.destination == "" && p.destinationPrefix == "" && request.Subject == "" && request.SubjectPrefix == "" {
		err = errors.Join(err, errors.New("must specify either subject or subject prefix"))
	}
	return err
}

// compile-time check for interface conformance
var _ Publisher = &publisher{}
