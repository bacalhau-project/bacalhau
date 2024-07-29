package ncl

import (
	"reflect"
	"strconv"
	"time"
)

/* Message Flow in NCL:

1. Creation: An application creates a Message with typed Payload and Metadata.

2. Serialization:
   a. MessageSerDe.Serialize converts the Message to a RawMessage (serializing the Payload).
   b. RawMessageSerDe.Serialize converts the RawMessage to a byte slice, adding envelope data (version, CRC, etc.).

3. Transmission: The serialized byte slice is sent over the network.

4. Reception: Another part of the system receives the byte slice.

5. Deserialization:
   a. RawMessageSerDe.Deserialize converts the byte slice to a RawMessage, handling envelope data.
   b. MessageSerDe.Deserialize converts the RawMessage to a Message, deserializing the Payload based on the provided type.

6. Processing: The application processes the fully deserialized Message.

This flow allows for efficient network transmission (using byte slices) and type-safe message handling (using Message),
while providing a clear separation between envelope handling (RawMessageSerDe) and payload processing (MessageSerDe).
*/

// Metadata keys
const (
	KeyMessageID       = "MessageID"
	KeyMessageType     = "Type"
	KeySource          = "Source"
	KeyEventTime       = "EventTime"
	KeyPayloadEncoding = "PayloadEncoding"
)

// Metadata contains metadata about the message
type Metadata map[string]string

// RawMessage represents a message after envelope handling but before payload deserialization.
// It contains metadata and a byte slice payload, which is the serialized form of the actual message content.
// RawMessage is the interface between RawMessageSerDe and MessageSerDe.
type RawMessage struct {
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

// NewMetadataFromMap creates a new shallow copy Metadata object from a map.
// Changes to the map will be reflected in the Metadata object, but more efficient than NewMetadataFromMapCopy
func NewMetadataFromMap(m map[string]string) *Metadata {
	if m == nil {
		return &Metadata{}
	}
	metadata := Metadata(m)
	return &metadata
}

// NewMetadataFromMapCopy creates a new deepcopy Metadata object from a map.
// Changes to the map will not be reflected in the Metadata object
func NewMetadataFromMapCopy(m map[string]string) *Metadata {
	metadata := make(Metadata, len(m))
	for k, v := range m {
		metadata[k] = v
	}
	return &metadata
}

// Get returns the value for a given key, or an empty string if the key doesn't exist
func (m Metadata) Get(key string) string {
	return m[key]
}

// Has checks if a key exists in the metadata
func (m Metadata) Has(key string) bool {
	_, ok := m[key]
	return ok
}

// Set sets the value for a given key
func (m Metadata) Set(key, value string) {
	m[key] = value
}

// SetInt sets the value for a given key as an int
func (m Metadata) SetInt(key string, value int) {
	m[key] = strconv.Itoa(value)
}

// SetInt64 sets the value for a given key as an int64
func (m Metadata) SetInt64(key string, value int64) {
	m[key] = strconv.FormatInt(value, 10)
}

// SetTime sets the value for a given key as a time.Time
func (m Metadata) SetTime(key string, value time.Time) {
	m.SetInt64(key, value.UnixNano())
}

// GetInt gets the value as an int, returning 0 if the key doesn't exist or the value isn't a valid int
func (m Metadata) GetInt(key string) int {
	if val, ok := m[key]; ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

// GetInt64 gets the value as an int64, returning 0 if the key doesn't exist or the value isn't a valid int64
func (m Metadata) GetInt64(key string) int64 {
	if val, ok := m[key]; ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// GetTime gets the value as a time.Time, returning the zero time if the key doesn't exist or the value isn't a valid time
func (m Metadata) GetTime(key string) time.Time {
	if val, ok := m[key]; ok {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return time.Unix(0, i)
		}
	}
	return time.Time{}
}

// ToMap returns the Metadata as a regular map[string]string
func (m Metadata) ToMap() map[string]string {
	return map[string]string(m)
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
