package ncl

import (
	"errors"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// ResponderConfig defines configuration for request handlers
type ResponderConfig struct {
	// Name identifies this responder instance
	Name string

	// MessageSerializer handles message envelope serialization
	// Optional: defaults to envelope.NewSerializer()
	MessageSerializer envelope.MessageSerializer

	// MessageRegistry for registering and deserializing message types
	MessageRegistry *envelope.Registry

	// Subject is the NATS subject to subscribe to
	Subject string

	// ProcessingTimeout is the maximum time allowed for processing a request
	// Optional: defaults to 5 seconds
	ProcessingTimeout time.Duration
}

const (
	DefaultResponderProcessingTimeout = 5 * time.Second
)

// DefaultResponderConfig returns a new ResponderConfig with default values
func DefaultResponderConfig() ResponderConfig {
	return ResponderConfig{
		MessageSerializer: envelope.NewSerializer(),
		ProcessingTimeout: DefaultResponderProcessingTimeout,
	}
}

// setDefaults applies default values to the config
func (c *ResponderConfig) setDefaults() {
	defaults := DefaultResponderConfig()
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
	if c.ProcessingTimeout == 0 {
		c.ProcessingTimeout = defaults.ProcessingTimeout
	}
}

// Validate checks if the config is valid
func (c *ResponderConfig) Validate() error {
	return errors.Join(
		validate.NotBlank(c.Name, "name cannot be blank"),
		validate.NotBlank(c.Subject, "subject cannot be blank"),
		validate.NotNil(c.MessageSerializer, "message serializer cannot be nil"),
		validate.NotNil(c.MessageRegistry, "message registry cannot be nil"),
		validate.IsGreaterThanZero(c.ProcessingTimeout, "processing timeout must be positive"),
	)
}
