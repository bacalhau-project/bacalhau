package envelope

import (
	"reflect"
)

// MessageSerializer handles low-level message serialization.
// It converts EncodedMessages to and from raw bytes.
type MessageSerializer interface {
	Serialize(*EncodedMessage) ([]byte, error)
	Deserialize([]byte) (*EncodedMessage, error)
}

// PayloadSerializer handles payload serialization within messages.
// It converts between typed Message and EncodedMessage formats.
type PayloadSerializer interface {
	Serialize(message *Message) (*EncodedMessage, error)
	Deserialize(rawMessage *EncodedMessage, payloadType reflect.Type) (*Message, error)
}
