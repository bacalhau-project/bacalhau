package envelope

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

// Registry manages the serialization and deserialization of the Payload field
// in Message structs. It simplifies payload handling in your application by providing:
//
// 1. Type Registration: Allows registering custom payload types with unique names.
// 2. Serialization Management: Handles serialization and deserialization of payloads using different encoding methods.
// 3. Type Resolution: Provides a mechanism to resolve between type names and their corresponding Go types.
//
// The Registry adds value to your application by:
//
//   - Automatic Payload Handling: Users can set any registered Go struct as the Message.Payload
//     without worrying about serialization. The manager handles this based on pre-configuration.
//
//   - Type Safety: By registering payload types, the system ensures that only known, expected
//     types are used as payloads, reducing runtime errors and improving system reliability.
//
//   - Flexibility: Supports multiple serialization formats for payloads, allowing different
//     message types to use the most appropriate format for their needs.
//
//   - Centralized Payload Type Management: Provides a single point of configuration for all
//     payload types used in the system, simplifying maintenance and reducing code duplication.
type Registry struct {
	nameToType  map[string]reflect.Type      // Maps payload type names to their reflect.Type
	typeToName  map[reflect.Type]string      // Maps reflect.Types to their registered names
	serializers map[string]PayloadSerializer // Maps encoding types to their respective PayloadSerializer
}

const (
	// JSONPayloadType is the identifier for JSON serialization/deserialization
	JSONPayloadType = "json"
	// DefaultPayloadEncoding is the default encoding used if none is specified
	DefaultPayloadEncoding = JSONPayloadType
)

// NewRegistry creates and initializes a new Registry
// It sets up the internal maps and registers the default JSON serializer
func NewRegistry() *Registry {
	return &Registry{
		nameToType: make(map[string]reflect.Type),
		typeToName: make(map[reflect.Type]string),
		serializers: map[string]PayloadSerializer{
			JSONPayloadType: &JSONPayloadSerializer{},
		},
	}
}

// Register adds a new payload type to the manager
// It registers both the name-to-type and type-to-name mappings
// Usage:
//
//	manager.Register("MyCustomType", MyCustomType{})
func (r *Registry) Register(name string, payload any) error {
	err := errors.Join(
		validate.NotBlank(name, "name cannot be blank"),
		validate.NotNil(payload, "payload cannot be nil"),
		validate.KeyNotInMap(name, r.nameToType, "name %s already registered", name),
	)

	if err != nil {
		return fmt.Errorf("failed to register payload: %w", err)
	}

	t := reflect.TypeOf(payload)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	r.nameToType[name] = t
	r.typeToName[t] = name
	return nil
}

// getType retrieves the reflect.Type for a given payload type name
// Returns an error if the type is not registered
func (r *Registry) getType(name string) (reflect.Type, error) {
	t, ok := r.nameToType[name]
	if !ok {
		return nil, NewErrUnsupportedMessageType(name)
	}
	return t, nil
}

// getName retrieves the registered name for a given payload instance
// Returns an error if the type is not registered
func (r *Registry) getName(payload any) (string, error) {
	t := reflect.TypeOf(payload)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name, ok := r.typeToName[t]
	if !ok {
		return "", NewErrUnsupportedMessageType(t.String())
	}
	return name, nil
}

// Serialize serializes a message using the specified serializer
// It handles default encoding, retrieves the correct serializer, and performs the serialization
// Usage:
//
//	rawMessage, err := manager.Serialize(message)
func (r *Registry) Serialize(message *Message) (*EncodedMessage, error) {
	// Set the default encoding if not specified
	if message.Metadata.Get(KeyPayloadEncoding) == "" {
		message.Metadata.Set(KeyPayloadEncoding, DefaultPayloadEncoding)
	}

	// Get the payload type name from the manager, and set it in the metadata
	payloadType, err := r.getName(message.Payload)
	if err != nil {
		return nil, err
	}
	message.Metadata.Set(KeyMessageType, payloadType)

	// Get the serializer for the specified encoding
	serializer, ok := r.serializers[message.Metadata.Get(KeyPayloadEncoding)]
	if !ok {
		return nil, NewErrUnsupportedEncoding(message.Metadata.Get(KeyPayloadEncoding))
	}

	// Perform the serialization
	rawMessage, err := serializer.Serialize(message)
	if err != nil {
		return nil, NewErrSerializationFailed(message.Metadata.Get(KeyPayloadEncoding), err)
	}
	return rawMessage, nil
}

// Deserialize deserializes a raw message using the specified deserializer
// It retrieves the correct deserializer, gets the payload type, and performs the deserialization
// Usage:
//
//	message, err := manager.Deserialize(rawMessage)
func (r *Registry) Deserialize(rawMessage *EncodedMessage) (*Message, error) {
	// Get the deserializer for the specified encoding
	deserializer, ok := r.serializers[rawMessage.Metadata.Get(KeyPayloadEncoding)]
	if !ok {
		return nil, NewErrUnsupportedEncoding(rawMessage.Metadata.Get(KeyPayloadEncoding))
	}

	// Get the payload type from the metadata
	payloadType, err := r.getType(rawMessage.Metadata.Get(KeyMessageType))
	if err != nil {
		return nil, err
	}

	// Perform the deserialization
	message, err := deserializer.Deserialize(rawMessage, payloadType)
	if err != nil {
		return nil, NewErrDeserializationFailed(rawMessage.Metadata.Get(KeyPayloadEncoding), err)
	}
	return message, nil
}
