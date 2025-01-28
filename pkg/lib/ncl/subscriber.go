package ncl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

// subscriber handles message consumption
type subscriber struct {
	nc      *nats.Conn
	config  SubscriberConfig
	encoder *encoder

	subscriptions       []*nats.Subscription
	consecutiveFailures int
	mu                  sync.Mutex
}

// NewSubscriber creates a new subscriber with the given options
func NewSubscriber(nc *nats.Conn, config SubscriberConfig) (Subscriber, error) {
	config.setDefaults()

	enc, err := newEncoder(encoderConfig{
		source:            config.Name,
		messageSerializer: config.MessageSerializer,
		messageRegistry:   config.MessageRegistry,
	})
	if err != nil {
		return nil, err
	}

	s := &subscriber{
		nc:      nc,
		config:  config,
		encoder: enc,
	}

	// Validate the subscriber
	if err := s.validate(); err != nil {
		return nil, fmt.Errorf("error validating subscriber: %w", err)
	}

	return s, nil
}

// validate checks if the subscriber is properly configured
func (s *subscriber) validate() error {
	return errors.Join(
		validate.NotNil(s.nc, "NATS connection cannot be nil"),
		s.config.Validate(),
	)
}

func (s *subscriber) Subscribe(ctx context.Context, subjects ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, subject := range subjects {
		sub, err := s.nc.Subscribe(subject, s.handleNatsMessage)
		if err != nil {
			return err
		}
		s.subscriptions = append(s.subscriptions, sub)
	}
	return nil
}

// messageHandler is the callback function for message processing
func (s *subscriber) handleNatsMessage(m *nats.Msg) {
	ctx := context.Background()
	metrics := telemetry.NewMetricRecorder(
		attribute.String(AttrInstance, s.config.Name),
		attribute.String(AttrOutcome, OutcomeSuccess),
	)
	var err error
	defer func() {
		if err != nil {
			metrics.Error(err)
			metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFailure))
		}
		metrics.Count(ctx, messageReceived)
		metrics.Done(ctx, messageProcessDuration)
	}()

	// Record message size
	metrics.Histogram(ctx, messageBytes, float64(len(m.Data)))

	// Process the message
	if err = s.processMessage(ctx, metrics, m); err != nil {
		log.Error().Err(err).Str("handler", s.config.Name).Msg("failed to process message")

		s.consecutiveFailures++
		delay := s.config.Backoff.BackoffDuration(s.consecutiveFailures)
		if nackErr := NackWithDelay(m, err, delay); nackErr != nil {
			log.Debug().Err(nackErr).Msg("failed to nack message")
		}
		return
	}

	s.consecutiveFailures = 0
	if ackErr := Ack(m); ackErr != nil {
		log.Warn().Err(ackErr).Msg("failed to ack message")
		metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeAckFailure))
	}
}

func (s *subscriber) processMessage(ctx context.Context, metrics *telemetry.MetricRecorder, m *nats.Msg) error {
	// TODO: interrupt processing if subscriber is closed
	ctx, cancel := context.WithTimeout(ctx, s.config.ProcessingTimeout)
	defer cancel()

	// Deserialize message envelope
	// Apply filter
	if s.config.MessageFilter.ShouldFilter(m.Header) {
		metrics.WithAttributes(attribute.String(AttrOutcome, OutcomeFiltered))
		return nil
	}
	metrics.Latency(ctx, messageProcessPartDuration, "filter")

	// Deserialize payload
	message, err := s.encoder.decode(m.Data)
	if err != nil {
		return fmt.Errorf("failed to deserialize message payload: %w", err)
	}
	metrics.Latency(ctx, messageProcessPartDuration, "decode")
	s.addMessageMetrics(ctx, metrics, message)

	// Process with handler
	if s.config.MessageHandler.ShouldProcess(ctx, message) {
		if err = s.config.MessageHandler.HandleMessage(ctx, message); err != nil {
			return fmt.Errorf("failed to handle message: %w", err)
		}
		metrics.Latency(ctx, messageProcessPartDuration, "handle")
	}

	// Notify successful processing
	s.config.ProcessingNotifier.OnProcessed(ctx, message)
	metrics.Latency(ctx, messageProcessPartDuration, "notify")

	return nil
}

// addMessageMetrics adds message metrics to the given metric recorder
func (s *subscriber) addMessageMetrics(ctx context.Context, metrics *telemetry.MetricRecorder, message *envelope.Message) {
	metrics.WithAttributes(
		attribute.String(AttrMessageType, message.Metadata.Get(envelope.KeyMessageType)),
		attribute.String(AttrSource, message.Metadata.Get(KeySource)),
	)

	// Calculate and record message latency if event time is present
	if eventTime := message.Metadata.GetTime(KeyEventTime); !eventTime.IsZero() {
		latency := time.Since(eventTime).Seconds()
		metrics.Histogram(ctx, messageLatency, latency)
	}
}

func (s *subscriber) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	for _, sub := range s.subscriptions {
		if sub.IsValid() {
			err = errors.Join(err, sub.Unsubscribe())
		}
	}
	if err != nil {
		return fmt.Errorf("error closing subscriptions: %w", err)
	}
	return nil
}

// compile-time interface assertions
var _ Subscriber = (*subscriber)(nil)
