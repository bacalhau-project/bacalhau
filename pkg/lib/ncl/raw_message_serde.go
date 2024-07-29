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

// JSONRawMessageSerializer handles serialization and deserialization using JSON
type JSONRawMessageSerializer struct{}

// Serialize encodes a RawMessage into a JSON message
func (j *JSONRawMessageSerializer) Serialize(msg *RawMessage) ([]byte, error) {
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
func (j *JSONRawMessageSerializer) Deserialize(data []byte) (*RawMessage, error) {
	if len(data) == 0 {
		return nil, errors.New(ErrEmptyData)
	}
	var serializableMsg serializableJSONMessage
	err := json.Unmarshal(data, &serializableMsg)
	if err != nil {
		return nil, err
	}

	return &RawMessage{
		Metadata: serializableMsg.Metadata,
		Payload:  serializableMsg.Payload,
	}, nil
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
func (p *ProtoSerializer) Deserialize(data []byte) (*RawMessage, error) {
	if len(data) == 0 {
		return nil, errors.New(ErrEmptyData)
	}
	var pbMsg v1.Message
	if err := proto.Unmarshal(data, &pbMsg); err != nil {
		return nil, err
	}

	return &RawMessage{
		Metadata: NewMetadataFromMap(pbMsg.Metadata.GetFields()),
		Payload:  pbMsg.Payload.Data,
	}, nil
}

// Compile time checks to ensure that serializers implement the RawMessageSerDe interface
var _ RawMessageSerDe = &JSONRawMessageSerializer{}
var _ RawMessageSerDe = &ProtoSerializer{}
