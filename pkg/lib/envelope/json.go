package envelope

import (
	"encoding/json"
	"errors"
	"reflect"
)

// JSONPayloadSerializer is a serializer/deserializer for JSON payloads
type JSONPayloadSerializer struct{}

// Serialize serializes a payload to JSON
func (j *JSONPayloadSerializer) Serialize(message *Message) (*EncodedMessage, error) {
	data, err := json.Marshal(message.Payload)
	if err != nil {
		return nil, err
	}
	return &EncodedMessage{
		Metadata: message.Metadata,
		Payload:  data,
	}, nil
}

// Deserialize deserializes a JSON payload
func (j *JSONPayloadSerializer) Deserialize(rMsg *EncodedMessage, payloadType reflect.Type) (*Message, error) {
	instance := reflect.New(payloadType).Interface()
	err := json.Unmarshal(rMsg.Payload, instance)
	if err != nil {
		return nil, err
	}
	return &Message{
		Metadata: rMsg.Metadata,
		Payload:  instance,
	}, nil
}

// serializableJSONMessage is used for JSON serialization
type serializableJSONMessage struct {
	Metadata *Metadata       `json:"metadata"`
	Payload  json.RawMessage `json:"payload"`
}

// JSONMessageSerializer handles serialization and deserialization using JSON
type JSONMessageSerializer struct{}

// Serialize encodes a envelope.EncodedMessage into a JSON message
func (j *JSONMessageSerializer) Serialize(msg *EncodedMessage) ([]byte, error) {
	if msg == nil {
		return nil, errors.New(ErrNilMessage)
	}

	// Create a serializableJSONMessage
	serializableMsg := serializableJSONMessage{
		Metadata: msg.Metadata,
		Payload:  json.RawMessage(msg.Payload),
	}

	return json.Marshal(serializableMsg)
}

// Deserialize decodes a JSON message into a envelope.EncodedMessage
func (j *JSONMessageSerializer) Deserialize(data []byte) (*EncodedMessage, error) {
	if len(data) == 0 {
		return nil, errors.New(ErrEmptyData)
	}
	var serializableMsg serializableJSONMessage
	err := json.Unmarshal(data, &serializableMsg)
	if err != nil {
		return nil, err
	}

	return &EncodedMessage{
		Metadata: serializableMsg.Metadata,
		Payload:  serializableMsg.Payload,
	}, nil
}

// Compile time check for interface conformance
var _ MessageSerializer = &JSONMessageSerializer{}
var _ PayloadSerializer = &JSONPayloadSerializer{}
