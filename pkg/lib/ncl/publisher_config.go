package ncl

import (
	"errors"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// AckMode determines how published messages should be acknowledged
type AckMode int

const (
	// ExplicitAck requires explicit acknowledgment from one subscriber
	ExplicitAck AckMode = iota

	// NoAck means the message is considered delivered as soon as it's published
	NoAck
)

// PublisherConfig defines configuration for a NATS publisher
type PublisherConfig struct {
	// Name identifies this publisher instance
	Name string

	// MessageSerializer handles message envelope serialization
	// Optional: defaults to envelope.NewSerializer()
	MessageSerializer envelope.MessageSerializer

	// MessageRegistry for registering and deserializing message types
	MessageRegistry *envelope.Registry

	// Either Destination or DestinationPrefix must be set, but not both

	// Destination is the exact NATS subject for all messages
	Destination string

	// DestinationPrefix is used to construct the subject by appending the message type
	// e.g. if prefix is "events", a UserCreated message type will be published to "events.UserCreated"
	DestinationPrefix string
}

func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		MessageSerializer: envelope.NewSerializer(),
	}
}

func (c *PublisherConfig) setDefaults() {
	defaults := DefaultPublisherConfig()
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
}

// Validate checks if the publisher options are properly configured
func (c *PublisherConfig) Validate() error {
	return errors.Join(
		validate.NotBlank(c.Name, "name cannot be blank"),
		validate.NotNil(c.MessageSerializer, "message serializer cannot be nil"),
		validate.NotNil(c.MessageRegistry, "message registry cannot be nil"),
		validate.False(c.Destination != "" && c.DestinationPrefix != "",
			"cannot specify both destination and destination prefix"),
	)
}

// OrderedPublisherConfig defines configuration for ordered publisher
type OrderedPublisherConfig struct {
	// Name identifies this publisher instance
	Name string

	// MessageSerializer handles message envelope serialization
	// Optional: defaults to envelope.NewSerializer()
	MessageSerializer envelope.MessageSerializer

	// MessageRegistry for registering and deserializing message types
	MessageRegistry *envelope.Registry

	// Either Destination or DestinationPrefix must be set, but not both

	// Destination is the exact NATS subject for all messages
	Destination string

	// DestinationPrefix is used to construct the subject by appending the message type
	// e.g. if prefix is "events", a UserCreated message type will be published to "events.UserCreated"
	DestinationPrefix string

	// AckWait is how long to wait for publish acknowledgement
	// Optional: defaults to 5s
	AckWait time.Duration

	// AckMode determines how messages should be acknowledged
	// Optional: defaults to ExplicitAck for backwards compatibility
	AckMode AckMode

	// MaxPending is the maximum number of queued messages
	// Optional: defaults to 1000
	MaxPending int

	// RetryAttempts is the number of publish retry attempts
	// Optional: defaults to 3
	RetryAttempts int

	// RetryWait is how long to wait between retry attempts
	// Optional: defaults to 1s
	RetryWait time.Duration
}

func DefaultOrderedPublisherConfig() OrderedPublisherConfig {
	return OrderedPublisherConfig{
		MessageSerializer: envelope.NewSerializer(),
		AckWait:           5 * time.Second,
		MaxPending:        1000,
		RetryAttempts:     3,
		RetryWait:         time.Second,
		AckMode:           ExplicitAck,
	}
}

func (c *OrderedPublisherConfig) setDefaults() {
	defaults := DefaultOrderedPublisherConfig()
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
	if c.AckWait == 0 {
		c.AckWait = defaults.AckWait
	}
	if c.MaxPending == 0 {
		c.MaxPending = defaults.MaxPending
	}
	if c.RetryAttempts == 0 {
		c.RetryAttempts = defaults.RetryAttempts
	}
	if c.RetryWait == 0 {
		c.RetryWait = defaults.RetryWait
	}
}

func (c *OrderedPublisherConfig) Validate() error {
	return errors.Join(
		validate.NotBlank(c.Name, "name cannot be blank"),
		validate.NotNil(c.MessageSerializer, "message serializer cannot be nil"),
		validate.NotNil(c.MessageRegistry, "message registry cannot be nil"),
		validate.False(c.Destination != "" && c.DestinationPrefix != "",
			"cannot specify both destination and destination prefix"),
		validate.IsGreaterThanZero(c.AckWait, "ack wait must be positive"),
		validate.IsGreaterThanZero(c.MaxPending, "max pending must be positive"),
		validate.IsGreaterThanZero(c.RetryAttempts, "retry attempts must be positive"),
		validate.IsGreaterThanZero(c.RetryWait, "retry wait must be positive"),
	)
}

func (c *OrderedPublisherConfig) toPublisherConfig() *PublisherConfig {
	return &PublisherConfig{
		Name:              c.Name,
		MessageSerializer: c.MessageSerializer,
		MessageRegistry:   c.MessageRegistry,
		Destination:       c.Destination,
		DestinationPrefix: c.DestinationPrefix,
	}
}
