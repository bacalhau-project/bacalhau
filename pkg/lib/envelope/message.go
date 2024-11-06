package envelope

import (
	"reflect"
)

/* Message Flow:

1. Creation: An application creates a Message with typed Payload and Metadata.

2. Serialization:
   a. PayloadSerializer.Serialize converts the Message to a EncodedMessage (serializing the Payload).
   b. MessageSerializer.Serialize converts the EncodedMessage to a byte slice, adding envelope data (version, CRC, etc.).

3. Transmission: The serialized byte slice is sent over the network.

4. Reception: Another part of the system receives the byte slice.

5. Deserialization:
   a. MessageSerializer.Deserialize converts the byte slice to a EncodedMessage, handling envelope data.
   b. PayloadSerializer.Deserialize converts the EncodedMessage to a Message, deserializing the Payload based on the provided type.

6. Processing: The application processes the fully deserialized Message.

This flow allows for efficient network transmission (using byte slices) and type-safe message handling (using Message),
while providing a clear separation between envelope handling (MessageSerializer) and payload processing (PayloadSerializer).
*/

// EncodedMessage represents a message after envelope handling but before payload deserialization.
// It contains metadata and a byte slice payload, which is the serialized form of the actual message content.
// EncodedMessage is the interface between MessageSerializer and PayloadSerializer.
type EncodedMessage struct {
	Metadata *Metadata
	Payload  []byte
}

// Message represents a fully deserialized message ready for processing by the application.
// It contains metadata and a deserialized payload of any type.
// Message is used by message handlers, filters, and subscribers.
type Message struct {
	Metadata *Metadata
	Payload  any
}

// NewMessage creates a new Message with the given payload
func NewMessage(payload any) *Message {
	return &Message{
		Metadata: &Metadata{},
		Payload:  payload,
	}
}

// WithMetadata sets the metadata for the message
func (m *Message) WithMetadata(metadata *Metadata) *Message {
	m.Metadata = metadata
	return m
}

// WithMetadataValue sets a key-value pair in the metadata
func (m *Message) WithMetadataValue(key, value string) *Message {
	m.Metadata.Set(key, value)
	return m
}

// IsType checks if the Message's Payload is of a specific type T
func (m *Message) IsType(t interface{}) bool {
	if m.Payload == nil {
		return false
	}

	payloadType := reflect.TypeOf(m.Payload)
	checkType := reflect.TypeOf(t)

	// If payload is a pointer, get its element type
	if payloadType.Kind() == reflect.Ptr {
		payloadType = payloadType.Elem()
	}

	// If check type is a pointer, get its element type
	if checkType.Kind() == reflect.Ptr {
		checkType = checkType.Elem()
	}

	return payloadType == checkType
}

// GetPayload retrieves the Payload as type T, handling both value and pointer types
func (m *Message) GetPayload(t interface{}) (interface{}, bool) {
	if m.Payload == nil {
		return nil, false
	}

	payloadType := reflect.TypeOf(m.Payload)
	checkType := reflect.TypeOf(t)

	// Remove pointer if present
	if payloadType.Kind() == reflect.Ptr {
		payloadType = payloadType.Elem()
	}
	checkValueType := checkType
	if checkType.Kind() == reflect.Ptr {
		checkValueType = checkType.Elem()
	}

	if payloadType != checkValueType {
		return nil, false
	}

	// If payload is a pointer but we're checking for a value, dereference it
	if reflect.TypeOf(m.Payload).Kind() == reflect.Ptr && checkType.Kind() != reflect.Ptr {
		return reflect.ValueOf(m.Payload).Elem().Interface(), true
	}

	// If payload is a value but we're checking for a pointer, return a pointer
	if reflect.TypeOf(m.Payload).Kind() != reflect.Ptr && checkType.Kind() == reflect.Ptr {
		val := reflect.ValueOf(m.Payload)
		ptr := reflect.New(val.Type())
		ptr.Elem().Set(val)
		return ptr.Interface(), true
	}

	return m.Payload, true
}
