package ncl

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// RequesterConfig defines configuration for a NATS publisher
type RequesterConfig struct {
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

func DefaultRequesterConfig() RequesterConfig {
	return RequesterConfig{
		MessageSerializer: envelope.NewSerializer(),
	}
}

func (c *RequesterConfig) setDefaults() {
	defaults := DefaultRequesterConfig()
	if c.MessageSerializer == nil {
		c.MessageSerializer = defaults.MessageSerializer
	}
}

// Validate checks if the publisher options are properly configured
func (c *RequesterConfig) Validate() error {
	return errors.Join(
		validate.NotBlank(c.Name, "name cannot be blank"),
		validate.NotNil(c.MessageSerializer, "message serializer cannot be nil"),
		validate.NotNil(c.MessageRegistry, "message registry cannot be nil"),
		validate.False(c.Destination != "" && c.DestinationPrefix != "",
			"cannot specify both destination and destination prefix"),
	)
}
