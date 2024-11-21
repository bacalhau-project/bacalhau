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
	nc     *nats.Conn
	config PublisherConfig
}

// NewPublisher creates a new publisher with the given options
func NewPublisher(nc *nats.Conn, config PublisherConfig) (Publisher, error) {
	config.setDefaults()

	p := &publisher{
		nc:     nc,
		config: config,
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
		p.config.Validate(),
	)
}

// Publish publishes a message to the NATS server
func (p *publisher) Publish(ctx context.Context, request PublishRequest) error {
	if err := p.validateRequest(request); err != nil {
		return err
	}

	msg, err := p.createMsg(request)
	if err != nil {
		return err
	}

	err = p.nc.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

func (p *publisher) validateRequest(request PublishRequest) error {
	err := errors.Join(
		validate.NotNil(request.Message, "cannot publish nil message"),
	)

	if request.Subject != "" && request.SubjectPrefix != "" {
		err = errors.Join(err, errors.New("cannot specify both subject and subject prefix"))
	}

	// if no p.destination or p.destinationPrefix is set, then the subject or subject prefix must be set
	if p.config.Destination == "" && p.config.DestinationPrefix == "" && request.Subject == "" && request.SubjectPrefix == "" {
		err = errors.Join(err, errors.New("must specify either subject or subject prefix"))
	}
	return err
}

// enrichMetadata adds metadata to the message, such as source and event time
func (p *publisher) enrichMetadata(message *envelope.Message) {
	message.Metadata.Set(KeySource, p.config.Name)
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
	if p.config.Destination != "" {
		return p.config.Destination
	}
	return fmt.Sprintf("%s.%s", p.config.DestinationPrefix, request.Message.Metadata.Get(envelope.KeyMessageType))
}

// createMsg processes a publish request and returns a nats message
func (p *publisher) createMsg(request PublishRequest) (*nats.Msg, error) {
	// Enrich metadata
	p.enrichMetadata(request.Message)

	// Serialize message
	rMsg, err := p.config.MessageRegistry.Serialize(request.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize into raw message: %w", err)
	}

	// Get subject
	subject := p.getSubject(request)

	// Serialize to bytes
	data, err := p.config.MessageSerializer.Serialize(rMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	return &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  request.Message.Metadata.ToHeaders(),
	}, nil
}

// compile-time check for interface conformance
var _ Publisher = &publisher{}
