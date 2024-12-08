package ncl

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// encoder handles all message serialization and deserialization
type encoder struct {
	serializer envelope.MessageSerializer
	registry   *envelope.Registry
	source     string
}

// encoderConfig contains configuration for encoder
type encoderConfig struct {
	// Name identifies the source of messages
	source string

	// MessageSerializer handles message envelope serialization
	// Optional: defaults to envelope.NewSerializer()
	messageSerializer envelope.MessageSerializer

	// MessageRegistry for registering and deserializing message types
	messageRegistry *envelope.Registry
}

func newEncoder(config encoderConfig) (*encoder, error) {
	errs := errors.Join(
		validate.NotBlank(config.source, "source cannot be blank"),
		validate.NotNil(config.messageSerializer, "message serializer cannot be nil"),
		validate.NotNil(config.messageRegistry, "message registry cannot be nil"),
	)
	if errs != nil {
		return nil, fmt.Errorf("invalid encoder configuration: %w", errs)
	}

	// Register error response type
	if err := config.messageRegistry.Register(ErrorMessageType, ErrorResponse{}); err != nil {
		if !strings.Contains(err.Error(), "already registered") {
			return nil, fmt.Errorf("failed to register error response type: %w", err)
		}
	}
	return &encoder{
		serializer: config.messageSerializer,
		registry:   config.messageRegistry,
		source:     config.source,
	}, nil
}

// encode converts a message to bytes and enriches metadata
func (m *encoder) encode(message *envelope.Message) ([]byte, error) {
	// Enrich metadata
	m.enrichMetadata(message)

	// Serialize message
	rMsg, err := m.registry.Serialize(message)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize into raw message: %w", err)
	}

	// Serialize to bytes
	data, err := m.serializer.Serialize(rMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	return data, nil
}

// decode converts bytes back to a message
func (m *encoder) decode(data []byte) (*envelope.Message, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("failed to deserialize message envelope: empty data")
	}

	// First try to deserialize as envelope
	rMsg, err := m.serializer.Deserialize(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize message envelope: %w", err)
	}

	// Then try to deserialize the payload
	message, err := m.registry.Deserialize(rMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize message payload: %w", err)
	}

	// Ensure times are in UTC when decoding
	if eventTime := message.Metadata.GetTime(KeyEventTime); !eventTime.IsZero() {
		message.Metadata.SetTime(KeyEventTime, eventTime.UTC())
	}

	return message, nil
}

// enrichMetadata adds common metadata to messages, ensuring consistent timezone usage
func (m *encoder) enrichMetadata(message *envelope.Message) {
	message.Metadata.Set(KeySource, m.source)
	if !message.Metadata.Has(KeyEventTime) {
		message.Metadata.SetTime(KeyEventTime, time.Now().UTC())
	} else {
		// Convert existing time to UTC if present
		existingTime := message.Metadata.GetTime(KeyEventTime)
		message.Metadata.SetTime(KeyEventTime, existingTime.UTC())
	}
}
