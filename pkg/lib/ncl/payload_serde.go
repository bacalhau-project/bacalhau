package ncl

import (
	"encoding/json"
	"reflect"
)

// JSONPayloadSerDe is a serializer/deserializer for JSON payloads
type JSONPayloadSerDe struct{}

// SerializePayload serializes a payload to JSON
func (j *JSONPayloadSerDe) SerializePayload(_ *Metadata, payload any) ([]byte, error) {
	return json.Marshal(payload)
}

// DeserializePayload deserializes a JSON payload
func (j *JSONPayloadSerDe) DeserializePayload(_ *Metadata, payloadType reflect.Type, data []byte) (any, error) {
	instance := reflect.New(payloadType).Interface()
	err := json.Unmarshal(data, instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// Compile time check for interface conformance
var _ PayloadSerDe = &JSONPayloadSerDe{}
