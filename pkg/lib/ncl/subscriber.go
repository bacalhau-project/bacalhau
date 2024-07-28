package ncl

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// subscriber handles message consumption
type subscriber struct {
	nc                  *nats.Conn
	messageHandlers     []MessageHandler
	messageFilter       MessageFilter
	checkpointer        Checkpointer
	messageDeserializer MessageSerDe
	payloadRegistry     *PayloadRegistry

	subscriptions []*nats.Subscription
	mu            sync.Mutex
}

// SubscriberOption is a function type for configuring a subscriber
type SubscriberOption func(*subscriber)

// WithSubscriberMessageHandlers sets the message handlers for the subscriber
func WithSubscriberMessageHandlers(handlers ...MessageHandler) SubscriberOption {
	return func(s *subscriber) {
		s.messageHandlers = append(s.messageHandlers, handlers...)
	}
}

// WithSubscriberCheckpointer sets the checkpointer for the subscriber
func WithSubscriberCheckpointer(checkpointer Checkpointer) SubscriberOption {
	return func(s *subscriber) {
		s.checkpointer = checkpointer
	}
}

// WithSubscriberMessageDeserializer sets the message deserializer for the subscriber
func WithSubscriberMessageDeserializer(deserializer MessageSerDe) SubscriberOption {
	return func(s *subscriber) {
		s.messageDeserializer = deserializer
	}
}

// WithSubscriberPayloadRegistry sets the payload registry for the subscriber
func WithSubscriberPayloadRegistry(registry *PayloadRegistry) SubscriberOption {
	return func(s *subscriber) {
		s.payloadRegistry = registry
	}
}

// WithSubscriberMessageFilter sets the message filter for the subscriber
func WithSubscriberMessageFilter(filter MessageFilter) SubscriberOption {
	return func(s *subscriber) {
		s.messageFilter = filter
	}
}

// defaultSubscriber returns a subscriber with default settings
func defaultSubscriber(nc *nats.Conn) *subscriber {
	return &subscriber{
		nc:                  nc,
		checkpointer:        &NoopCheckpointer{},
		messageHandlers:     []MessageHandler{},
		messageFilter:       NoopMessageFilter{},
		messageDeserializer: NewEnvelopeSerializer(),
		subscriptions:       []*nats.Subscription{},
		mu:                  sync.Mutex{},
	}
}

// NewSubscriber creates a new subscriber with the given options
func NewSubscriber(nc *nats.Conn, opts ...SubscriberOption) (Subscriber, error) {
	// Start with default subscriber
	s := defaultSubscriber(nc)

	// Apply all options
	for _, opt := range opts {
		opt(s)
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
		validate.NotNil(s.checkpointer, "checkpointer cannot be nil"),
		validate.NotNil(s.messageDeserializer, "message deserializer cannot be nil"),
		validate.NotNil(s.payloadRegistry, "payload registry cannot be nil"),
		validate.IsNotEmpty(s.messageHandlers, "message handlers cannot be empty"),
		validate.NotNil(s.messageFilter, "message filter cannot be nil"),
	)
}

// Subscribe starts consuming messages from the given subjects
func (s *subscriber) Subscribe(subjects ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, subject := range subjects {
		sub, err := s.nc.Subscribe(subject, s.processMessage)
		if err != nil {
			return err
		}
		s.subscriptions = append(s.subscriptions, sub)
	}
	return nil
}

func (s *subscriber) processMessage(m *nats.Msg) {
	// TODO: implement a better context
	ctx := context.Background()
	rMsg := &RawMessage{}
	if err := s.messageDeserializer.Deserialize(m.Data, rMsg); err != nil {
		// Handle error
		return
	}

	// Apply filter
	if s.messageFilter.ShouldFilter(rMsg.Metadata) {
		return
	}

	// Deserialize payload
	payload, err := s.payloadRegistry.DeserializePayload(rMsg.Metadata, rMsg.Payload)
	if err != nil {
		// Handle error
		return
	}

	message := &Message{
		Metadata: rMsg.Metadata,
		Payload:  payload,
	}

	for _, handler := range s.messageHandlers {
		if handler.ShouldProcess(ctx, message) {
			if err = handler.HandleMessage(ctx, message); err != nil {
				// Handle error
				return
			}
		}
	}
	if err = s.checkpointer.Checkpoint(message); err != nil {
		// Handle error
		return
	}
}

// Close closes the subscriber
func (s *subscriber) Close(_ context.Context) error {
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
