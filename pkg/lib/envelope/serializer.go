package envelope

import (
	"encoding/binary"
	"hash/crc32"
)

// Error message constants
const (
	ErrMsgTooShort  = "too short"
	ErrMsgCRCFailed = "CRC check failed"
)

const (
	VersionSize = 1
	CRCSize     = 4
	HeaderSize  = VersionSize + CRCSize
)

// Serializer handles the serialization and deserialization of messages
// with version information and CRC checks. It wraps the actual message serialization
// with additional metadata for versioning and integrity checking.
//
// Envelope Structure:
// +----------------+----------------+--------------------+
// | Version (1 byte)| CRC (4 bytes) | Serialized envelope.Message |
// +----------------+----------------+--------------------+
//
// - Version: Indicates the schema version used for serialization (1 byte)
// - CRC: A 32-bit CRC checksum of the serialized message (4 bytes)
// - Serialized envelope.Message: The actual message content, serialized by a version-specific serializer
//
// The Serializer adds a version byte and a CRC checksum to each serialized message,
// allowing for future extensibility, backward compatibility, and data integrity verification.
type Serializer struct {
	// serializationVersion represents the current schema version used for serializing messages.
	// This version is included in the envelope header of each serialized message.
	// Example: SchemaVersionJSONV1 for JSON serialization
	serializationVersion SchemaVersion
	// serializers is a map of schema versions to their corresponding MessageSerializer implementations.
	// It allows the Serializer to:
	// 1. Use different serialization methods based on the current schema version.
	// 2. Deserialize messages that were encoded using different schema versions.
	// This enables backward compatibility and supports evolution of the serialization format.
	//
	// Example: {
	//    SchemaVersionJSONV1:     &JSONMessageSerializer{},
	//    SchemaVersionProtobufV1: &ProtoMessageSerializer{},
	// }
	serializers map[SchemaVersion]MessageSerializer
}

// NewSerializer creates a new Serializer with default serializers
func NewSerializer() *Serializer {
	return &Serializer{
		serializationVersion: DefaultSchemaVersion,
		serializers: map[SchemaVersion]MessageSerializer{
			SchemaVersionJSONV1:     &JSONMessageSerializer{},
			SchemaVersionProtobufV1: &ProtoMessageSerializer{},
		},
	}
}

// WithSerializationVersion sets the schema version used for serialization.
// This version will be used for all subsequent Serialize calls.
// It does not affect the deserialization of messages.
func (v *Serializer) WithSerializationVersion(version SchemaVersion) *Serializer {
	v.serializationVersion = version
	return v
}

// Serialize encodes a envelope.EncodedMessage into a byte slice, adding version information
// and a CRC checksum. It uses the serializer corresponding to the current serializationVersion.
func (v *Serializer) Serialize(msg *EncodedMessage) ([]byte, error) {
	serializer := v.serializers[v.serializationVersion]
	msgBytes, err := serializer.Serialize(msg)
	if err != nil {
		return nil, NewErrSerializationFailed(v.serializationVersion.String(), err)
	}

	// Allocate the final message buffer
	finalMsg := make([]byte, HeaderSize+len(msgBytes))

	// Set SchemaVersion
	finalMsg[0] = byte(v.serializationVersion)

	// Copy serialized message
	copy(finalMsg[HeaderSize:], msgBytes)

	// Calculate and set CRC-32 of the serialized message
	crc := crc32.ChecksumIEEE(finalMsg[HeaderSize:])
	binary.BigEndian.PutUint32(finalMsg[VersionSize:HeaderSize], crc)

	return finalMsg, nil
}

// Deserialize decodes a byte slice into a envelope.EncodedMessage. It verifies the schema version
// and CRC checksum before using the appropriate deserializer to decode the message.
func (v *Serializer) Deserialize(data []byte) (*EncodedMessage, error) {
	if len(data) < HeaderSize {
		return nil, NewErrBadMessage(ErrMsgTooShort)
	}

	version := SchemaVersion(data[0])
	deserializer, ok := v.serializers[version]
	if !ok {
		return nil, NewErrUnsupportedEncoding(version.String())
	}

	// Verify CRC
	expectedCRC := binary.BigEndian.Uint32(data[VersionSize:HeaderSize])
	actualCRC := crc32.ChecksumIEEE(data[HeaderSize:])
	if actualCRC != expectedCRC {
		return nil, NewErrBadMessage(ErrMsgCRCFailed)
	}

	msg, err := deserializer.Deserialize(data[HeaderSize:])
	if err != nil {
		return nil, NewErrDeserializationFailed(version.String(), err)
	}
	return msg, nil
}

// Compile time checks to ensure that serializers implement the MessageSerializer interface
var _ MessageSerializer = &Serializer{}
