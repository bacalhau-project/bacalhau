package ncl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// publisher handles message publishing
type publisher struct {
	nc                *nats.Conn
	messageSerializer MessageSerDe
	payloadRegistry   *PayloadRegistry
	name              string
	destination       string
	destinationPrefix string
}

// PublisherOption is a function type for configuring a publisher
type PublisherOption func(*publisher)

// WithPublisherMessageSerializer sets the message serializer for the publisher
func WithPublisherMessageSerializer(serializer MessageSerDe) PublisherOption {
	return func(p *publisher) {
		p.messageSerializer = serializer
	}
}

// WithPublisherPayloadRegistry sets the payload registry for the publisher
func WithPublisherPayloadRegistry(registry *PayloadRegistry) PublisherOption {
	return func(p *publisher) {
		p.payloadRegistry = registry
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
		messageSerializer: NewEnvelopeSerializer(),
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
	err := errors.Join(
		validate.NotNil(p.nc, "NATS connection cannot be nil"),
		validate.NotBlank(p.name, "publisher name cannot be blank"),
		validate.NotNil(p.messageSerializer, "message serializer cannot be nil"),
		validate.NotNil(p.payloadRegistry, "payload registry cannot be nil"),
	)

	// Check one of destination or destinationPrefix is set
	if p.destination == "" && p.destinationPrefix == "" {
		err = errors.Join(err, errors.New("destination or destination prefix must be set"))
	} else if p.destination != "" && p.destinationPrefix != "" {
		err = errors.Join(err, errors.New("destination and destination prefix cannot both be set"))
	}
	return err
}

// Publish publishes a message to the NATS server
func (p *publisher) Publish(ctx context.Context, event any) error {
	return p.PublishWithMetadata(ctx, &Metadata{}, event)
}

// PublishWithMetadata publishes a message to the NATS server with metadata
func (p *publisher) PublishWithMetadata(_ context.Context, metadata *Metadata, event any) error {
	p.enrichMetadata(metadata)
	msg, err := p.constructMessage(metadata, event)
	if err != nil {
		return fmt.Errorf("failed to construct message: %w", err)
	}

	subject := p.getSubject(metadata)

	data, err := p.messageSerializer.Serialize(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	err = p.nc.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

func (p *publisher) constructMessage(metadata *Metadata, event any) (*RawMessage, error) {
	payload, err := p.payloadRegistry.SerializePayload(metadata, event)
	if err != nil {
		return nil, err
	}

	return &RawMessage{
		Metadata: metadata,
		Payload:  payload,
	}, nil
}

// enrichMetadata adds the publisher name to the metadata
func (p *publisher) enrichMetadata(metadata *Metadata) {
	metadata.Set(KeySource, p.name)
	if !metadata.Has(KeyEventTime) {
		metadata.SetTime(KeyEventTime, time.Now())
	}
}

func (p *publisher) getSubject(metadata *Metadata) string {
	if p.destination != "" {
		return p.destination
	}
	return fmt.Sprintf("%s.%s", p.destinationPrefix, metadata.Get(KeyMessageType))
}

// compile-time check for interface conformance
var _ Publisher = &publisher{}
