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

// requester handles message publishing
type requester struct {
	nc     *nats.Conn
	config RequesterConfig
}

// NewRequester creates a new requester with the given options
func NewRequester(nc *nats.Conn, config RequesterConfig) (Requester, error) {
	config.setDefaults()

	p := &requester{
		nc:     nc,
		config: config,
	}

	// Validate the requester
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("error validating requester: %w", err)
	}

	return p, nil
}

// validate checks if the requester is properly configured
func (p *requester) validate() error {
	return errors.Join(
		validate.NotNil(p.nc, "NATS connection cannot be nil"),
		p.config.Validate(),
	)
}

func (p *requester) Request(ctx context.Context, request PublishRequest) (*envelope.Message, error) {
	if err := p.validateRequest(request); err != nil {
		return nil, err
	}

	msg, err := p.createMsg(request)
	if err != nil {
		return nil, err
	}

	resp, err := p.nc.RequestMsgWithContext(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to request message: %w", err)
	}

	// Deserialize message envelope
	rMsg, err := p.config.MessageSerializer.Deserialize(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize message envelope: %w", err)
	}

	// Deserialize payload
	message, err := p.config.MessageRegistry.Deserialize(rMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize message: %w", err)
	}

	// return error if message is an error response
	if errorResponse, ok := message.GetPayload(ErrorResponse{}); ok {
		return nil, errorResponse.(*ErrorResponse)
	}

	return message, nil
}

func (p *requester) validateRequest(request PublishRequest) error {
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
func (p *requester) enrichMetadata(message *envelope.Message) {
	message.Metadata.Set(KeySource, p.config.Name)
	if !message.Metadata.Has(KeyEventTime) {
		message.Metadata.SetTime(KeyEventTime, time.Now())
	}
}

func (p *requester) getSubject(request PublishRequest) string {
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
func (p *requester) createMsg(request PublishRequest) (*nats.Msg, error) {
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
var _ Requester = &requester{}
