package envelope

import (
	"errors"

	"google.golang.org/protobuf/proto"

	v1 "github.com/bacalhau-project/bacalhau/pkg/lib/envelope/proto/v1"
)

// ProtoMessageSerializer handles serialization and deserialization using Protocol Buffers v1
type ProtoMessageSerializer struct{}

// Serialize encodes a envelope.EncodedMessage into a Protocol Buffers message
func (p *ProtoMessageSerializer) Serialize(msg *EncodedMessage) ([]byte, error) {
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

// Deserialize decodes a Protocol Buffers message into a envelope.EncodedMessage
func (p *ProtoMessageSerializer) Deserialize(data []byte) (*EncodedMessage, error) {
	if len(data) == 0 {
		return nil, errors.New(ErrEmptyData)
	}
	var pbMsg v1.Message
	if err := proto.Unmarshal(data, &pbMsg); err != nil {
		return nil, err
	}

	return &EncodedMessage{
		Metadata: NewMetadataFromMap(pbMsg.Metadata.GetFields()),
		Payload:  pbMsg.Payload.Data,
	}, nil
}

// Compile time checks to ensure that serializers implement the MessageSerializer interface
var _ MessageSerializer = &ProtoMessageSerializer{}
