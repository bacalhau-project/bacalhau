package ncl

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/proto"

	v1 "github.com/bacalhau-project/bacalhau/pkg/lib/ncl/proto/v1"
)

const (
	ErrNilMessage = "message is nil"
	ErrEmptyData  = "data is empty"
)

// serializableJSONMessage is used for JSON serialization
type serializableJSONMessage struct {
	Metadata *Metadata       `json:"metadata"`
	Payload  json.RawMessage `json:"payload"`
}

// JSONMessageSerializer handles serialization and deserialization using JSON
type JSONMessageSerializer struct{}

// Serialize encodes a RawMessage into a JSON message
func (j *JSONMessageSerializer) Serialize(msg *RawMessage) ([]byte, error) {
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

// Deserialize decodes a JSON message into a RawMessage
func (j *JSONMessageSerializer) Deserialize(data []byte, msg *RawMessage) error {
	if len(data) == 0 {
		return errors.New(ErrEmptyData)
	}
	var serializableMsg serializableJSONMessage
	err := json.Unmarshal(data, &serializableMsg)
	if err != nil {
		return err
	}

	msg.Metadata = serializableMsg.Metadata
	msg.Payload = serializableMsg.Payload
	return nil
}

// ProtoSerializer handles serialization and deserialization using Protocol Buffers v1
type ProtoSerializer struct{}

// Serialize encodes a RawMessage into a Protocol Buffers message
func (p *ProtoSerializer) Serialize(msg *RawMessage) ([]byte, error) {
	if msg == nil {
		return nil, errors.New(ErrNilMessage)
	}
	pbMsg := &v1.Message{
		Metadata: &v1.Metadata{
			Fields: msg.Metadata.ToMap(),
		},
		Payload: &v1.Payload{
			Data: msg.Payload,
		},
	}
	return proto.Marshal(pbMsg)
}

// Deserialize decodes a Protocol Buffers message into a RawMessage
func (p *ProtoSerializer) Deserialize(data []byte, msg *RawMessage) error {
	if len(data) == 0 {
		return errors.New(ErrEmptyData)
	}
	var pbMsg v1.Message
	if err := proto.Unmarshal(data, &pbMsg); err != nil {
		return err
	}

	msg.Metadata = NewMetadataFromMap(pbMsg.Metadata.GetFields())
	msg.Payload = pbMsg.Payload.Data

	return nil
}

// Compile time checks to ensure that serializers implement the MessageSerDe interface
var _ MessageSerDe = &JSONMessageSerializer{}
var _ MessageSerDe = &ProtoSerializer{}
