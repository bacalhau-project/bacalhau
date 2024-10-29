package watcher

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// jsonEvent represents the JSON structure of an Event for serialization purposes
type jsonEvent struct {
	SeqNum     uint64          `json:"seqNum"`
	Operation  Operation       `json:"operation"`
	ObjectType string          `json:"objectType"`
	Object     json.RawMessage `json:"object"`
	Timestamp  int64           `json:"timestamp"`
}

// JSONSerializer implements the Serializer interface for JSON serialization
type JSONSerializer struct {
	typeRegistry map[string]reflect.Type
}

// NewJSONSerializer creates a new instance of JSONSerializer
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{
		typeRegistry: make(map[string]reflect.Type),
	}
}

// RegisterType adds a new type to the serializer's type registry
// It returns an error if the type is already registered or if the provided type is invalid
func (s *JSONSerializer) RegisterType(name string, t reflect.Type) error {
	if _, exists := s.typeRegistry[name]; exists {
		return fmt.Errorf("type %s is already registered", name)
	}
	if t == nil {
		return fmt.Errorf("cannot register nil type for %s", name)
	}
	if t.Kind() == reflect.Ptr {
		return fmt.Errorf("cannot register pointer type for %s, use base type instead", name)
	}
	s.typeRegistry[name] = t
	return nil
}

// IsTypeRegistered checks if a type is registered in the serializer's type registry
func (s *JSONSerializer) IsTypeRegistered(name string) bool {
	_, exists := s.typeRegistry[name]
	return exists
}

// Marshal serializes an Event into a byte slice
func (s *JSONSerializer) Marshal(v Event) ([]byte, error) {
	// Serialize the Object field separately to handle interface{} properly
	objectJson, err := json.Marshal(v.Object)
	if err != nil {
		return nil, NewSerializationError(v, err)
	}

	// Create a jsonEvent struct and populate it
	jEvent := jsonEvent{
		SeqNum:     v.SeqNum,
		Operation:  v.Operation,
		ObjectType: v.ObjectType,
		Object:     objectJson,
		Timestamp:  v.Timestamp.UnixNano(),
	}

	// Marshal the entire jsonEvent struct
	data, err := json.Marshal(jEvent)
	if err != nil {
		return nil, NewSerializationError(v, err)
	}
	return data, nil
}

// Unmarshal deserializes a byte slice into an Event
func (s *JSONSerializer) Unmarshal(data []byte, event *Event) error {
	jEvent := new(jsonEvent)
	err := json.Unmarshal(data, jEvent)
	if err != nil {
		return NewDeserializationError(err)
	}

	// Populate the Event struct with the deserialized data
	event.SeqNum = jEvent.SeqNum
	event.Operation = jEvent.Operation
	event.ObjectType = jEvent.ObjectType
	event.Timestamp = time.Unix(0, jEvent.Timestamp).UTC()

	// Look up the registered type for this object
	t, ok := s.typeRegistry[jEvent.ObjectType]
	if !ok {
		return NewDeserializationError(fmt.Errorf("unknown object type: %s", jEvent.ObjectType))
	}

	event.Object, err = s.castObject(jEvent.Object, t)
	if err != nil {
		return err
	}
	return nil
}

// castObject attempts to cast an object to the specified type
func (s *JSONSerializer) castObject(obj json.RawMessage, t reflect.Type) (interface{}, error) {
	if obj == nil || len(obj) == 0 || string(obj) == "null" {
		return nil, nil
	}

	// Create a new instance of the registered type
	v := reflect.New(t).Interface()
	err := json.Unmarshal(obj, v)
	if err != nil {
		return nil, NewDeserializationError(fmt.Errorf("failed to unmarshal object type: %s %w", t.Name(), err))
	}

	return reflect.ValueOf(v).Elem().Interface(), nil // Get the actual value, not the pointer
}

// compile time check for interface conformance
var _ Serializer = &JSONSerializer{}
