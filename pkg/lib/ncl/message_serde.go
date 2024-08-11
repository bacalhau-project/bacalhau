package ncl

import (
	"encoding/json"
	"reflect"
)

// JSONMessageSerDe is a serializer/deserializer for JSON payloads
type JSONMessageSerDe struct{}

// Serialize serializes a payload to JSON
func (j *JSONMessageSerDe) Serialize(message *Message) (*RawMessage, error) {
	data, err := json.Marshal(message.Payload)
	if err != nil {
		return nil, err
	}
	return &RawMessage{
		Metadata: message.Metadata,
		Payload:  data,
	}, nil
}

// Deserialize deserializes a JSON payload
func (j *JSONMessageSerDe) Deserialize(rMsg *RawMessage, payloadType reflect.Type) (*Message, error) {
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

// Compile time check for interface conformance
var _ MessageSerDe = &JSONMessageSerDe{}
