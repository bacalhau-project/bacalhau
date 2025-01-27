package ncl

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

// publisher handles message publishing
type publisher struct {
	nc      *nats.Conn
	config  PublisherConfig
	encoder *encoder
}

// NewPublisher creates a new publisher that can handle both publish and request operations
func NewPublisher(nc *nats.Conn, config PublisherConfig) (Publisher, error) {
	config.setDefaults()

	enc, err := newEncoder(encoderConfig{
		source:            config.Name,
		messageSerializer: config.MessageSerializer,
		messageRegistry:   config.MessageRegistry,
	})
	if err != nil {
		return nil, err
	}

	p := &publisher{
		nc:      nc,
		config:  config,
		encoder: enc,
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

// Publish sends a message without expecting a response
func (p *publisher) Publish(ctx context.Context, request PublishRequest) error {
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrInstance, p.config.Name),
		attribute.String(AttrOutcome, OutcomeSuccess),
	)
	var err error
	defer func() {
		if err != nil {
			metrics.Error(err)
			metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFailure))
		}
		metrics.Count(ctx, publishCount)
		metrics.Done(ctx, publishDuration)
	}()

	if err = p.validateRequest(request); err != nil {
		return err
	}

	msg, err := p.encodeMsg(request)
	if err != nil {
		return err
	}
	metrics.Latency(ctx, publishPartDuration, "encode")
	metrics.Histogram(ctx, publishBytes, float64(len(msg.Data)))
	p.addMessageMetrics(ctx, metrics, msg, request.Message)

	if err = p.nc.PublishMsg(msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	metrics.Latency(ctx, publishPartDuration, "publish")

	return nil
}

// Request sends a message and waits for a response
func (p *publisher) Request(ctx context.Context, request PublishRequest) (*envelope.Message, error) {
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrInstance, p.config.Name),
		attribute.String(AttrOutcome, OutcomeSuccess),
	)
	var err error
	defer func() {
		if err != nil {
			metrics.Error(err)
			metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFailure))
		}
		metrics.Count(ctx, requesterCount)
		metrics.Done(ctx, requesterDuration)
	}()

	if err = p.validateRequest(request); err != nil {
		return nil, err
	}

	msg, err := p.encodeMsg(request)
	if err != nil {
		return nil, err
	}
	metrics.Latency(ctx, requesterPartDuration, "encode")
	metrics.Histogram(ctx, requesterBytes, float64(len(msg.Data)))
	p.addMessageMetrics(ctx, metrics, msg, request.Message)

	// Use context for timeout
	resp, err := p.nc.RequestMsgWithContext(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to request message: %w", err)
	}
	metrics.Latency(ctx, requesterPartDuration, "request")
	metrics.Histogram(ctx, requesterResponseBytes, float64(len(resp.Data)))

	// Deserialize response
	message, err := p.encoder.decode(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}
	metrics.Latency(ctx, requesterPartDuration, "decode")

	// Check if response is an error
	if errorResponse, ok := message.GetPayload(bacerrors.New("")); ok {
		err = errorResponse.(bacerrors.Error)
		return nil, err
	}

	return message, nil
}

// validateRequest validates the publish request
func (p *publisher) validateRequest(request PublishRequest) error {
	err := errors.Join(
		validate.NotNil(request.Message, "cannot publish nil message"),
	)

	if request.Subject != "" && request.SubjectPrefix != "" {
		err = errors.Join(err, errors.New("cannot specify both subject and subject prefix"))
	}

	// if no p.destination or p.destinationPrefix is set, then the subject or subject prefix must be set
	if p.config.Destination == "" && p.config.DestinationPrefix == "" &&
		request.Subject == "" && request.SubjectPrefix == "" {
		err = errors.Join(err, errors.New("must specify either subject or subject prefix"))
	}
	return err
}

// getSubject determines the subject to publish to
func (p *publisher) getSubject(request PublishRequest) string {
	if request.Subject != "" {
		return request.Subject
	}
	if request.SubjectPrefix != "" {
		return fmt.Sprintf("%s.%s", request.SubjectPrefix,
			request.Message.Metadata.Get(envelope.KeyMessageType))
	}
	if p.config.Destination != "" {
		return p.config.Destination
	}
	return fmt.Sprintf("%s.%s", p.config.DestinationPrefix,
		request.Message.Metadata.Get(envelope.KeyMessageType))
}

// encodeMsg creates a NATS message from the request
func (p *publisher) encodeMsg(request PublishRequest) (*nats.Msg, error) {
	data, err := p.encoder.encode(request.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	// Get subject
	subject := p.getSubject(request)

	return &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  request.Message.Metadata.ToHeaders(),
	}, nil
}

func (p *publisher) addMessageMetrics(ctx context.Context, metrics *telemetry.MetricRecorder, msg *nats.Msg, message *envelope.Message) {
	metrics.WithAttributes(
		attribute.String(AttrMessageType, message.Metadata.Get(envelope.KeyMessageType)),
		attribute.String(AttrSource, message.Metadata.Get(KeySource)),
	)
}

// compile-time check for interface conformance
var _ Publisher = &publisher{}
