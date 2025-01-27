package ncl

import (
	"errors"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// SubscriberConfig defines configuration for a NATS subscriber
type SubscriberConfig struct {
	// Name identifies this subscriber instance
	Name string

	// MessageSerializer handles message envelope serialization
	// Optional: defaults to envelope.NewSerializer()
	MessageSerializer envelope.MessageSerializer

	// MessageRegistry for registering and deserializing message types
	MessageRegistry *envelope.Registry

	// MessageHandler processes received messages
	MessageHandler MessageHandler

	// MessageFilter determines which messages should be processed
	// Optional: defaults to NoopMessageFilter which processes all messages
	MessageFilter MessageFilter

	// ProcessingNotifier is notified when messages are successfully processed
	// Optional: defaults to NoopNotifier
	ProcessingNotifier ProcessingNotifier

	// ProcessingTimeout is the maximum time allowed for processing a message
	// Optional: defaults to 5 seconds
	ProcessingTimeout time.Duration

	// Backoff strategy for failed message processing
	// Optional: defaults to exponential backoff
	Backoff backoff.Backoff
}

const (
	DefaultProcessingTimeout   = 30 * time.Second
	DefaultBackoffInitialDelay = 100 * time.Millisecond
	DefaultBackoffMaximumDelay = 5 * time.Second
)

func DefaultSubscriberConfig() SubscriberConfig {
	return SubscriberConfig{
		MessageSerializer:  envelope.NewSerializer(),
		MessageFilter:      &NoopMessageFilter{},
		ProcessingNotifier: &NoopNotifier{},
		ProcessingTimeout:  DefaultProcessingTimeout,
		Backoff: backoff.NewExponential(
			DefaultBackoffInitialDelay,
			DefaultBackoffMaximumDelay,
		),
	}
}

func (c *SubscriberConfig) setDefaults() {
	defaults := DefaultSubscriberConfig()
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
	if c.MessageFilter == nil {
		c.MessageFilter = defaults.MessageFilter
	}
	if c.ProcessingNotifier == nil {
		c.ProcessingNotifier = defaults.ProcessingNotifier
	}
	if c.ProcessingTimeout == 0 {
		c.ProcessingTimeout = defaults.ProcessingTimeout
	}
	if c.Backoff == nil {
		c.Backoff = defaults.Backoff
	}
}

func (c *SubscriberConfig) Validate() error {
	return errors.Join(
		validate.NotBlank(c.Name, "name cannot be blank"),
		validate.NotNil(c.MessageSerializer, "message serializer cannot be nil"),
		validate.NotNil(c.MessageRegistry, "message registry cannot be nil"),
		validate.NotNil(c.MessageHandler, "message handler cannot be nil"),
		validate.NotNil(c.MessageFilter, "message filter cannot be nil"),
		validate.NotNil(c.ProcessingNotifier, "processing notifier cannot be nil"),
		validate.IsGreaterThanZero(c.ProcessingTimeout, "processing timeout must be positive"),
		validate.NotNil(c.Backoff, "backoff cannot be nil"),
	)
}
