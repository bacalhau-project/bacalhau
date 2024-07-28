package ncl

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

// EnvelopeSerializer handles the serialization and deserialization of messages
// with version information and CRC checks. It wraps the actual message serialization
// with additional metadata for versioning and integrity checking.
//
// Envelope Structure:
// +----------------+----------------+--------------------+
// | Version (1 byte)| CRC (4 bytes) | Serialized Message |
// +----------------+----------------+--------------------+
//
// - Version: Indicates the schema version used for serialization (1 byte)
// - CRC: A 32-bit CRC checksum of the serialized message (4 bytes)
// - Serialized Message: The actual message content, serialized by a version-specific serializer
//
// The EnvelopeSerializer adds a version byte and a CRC checksum to each serialized message,
// allowing for future extensibility, backward compatibility, and data integrity verification.
type EnvelopeSerializer struct {
	serializerSchemaVersion SchemaVersion
	serializers             map[SchemaVersion]MessageSerDe
}

// NewEnvelopeSerializer creates a new EnvelopeSerializer with default serializers
func NewEnvelopeSerializer() *EnvelopeSerializer {
	return &EnvelopeSerializer{
		serializerSchemaVersion: DefaultSchemaVersion,
		serializers: map[SchemaVersion]MessageSerDe{
			SchemaVersionJSONV1:     &JSONMessageSerializer{},
			SchemaVersionProtobufV1: &ProtoSerializer{},
		},
	}
}

// WithSchemaVersion sets the schema version used for serialization of new messages.
// it does not affect the deserialization of messages.
func (v *EnvelopeSerializer) WithSchemaVersion(version SchemaVersion) *EnvelopeSerializer {
	v.serializerSchemaVersion = version
	return v
}

// Serialize encodes a RawMessage into a byte slice, adding version information
// and a CRC checksum. It uses the appropriate serializer based on the schema version.
func (v *EnvelopeSerializer) Serialize(msg *RawMessage) ([]byte, error) {
	version := v.serializerSchemaVersion
	serializer := v.serializers[version]
	msgBytes, err := serializer.Serialize(msg)
	if err != nil {
		return nil, NewErrSerializationFailed(version.String(), err)
	}

	// Allocate the final message buffer
	finalMsg := make([]byte, HeaderSize+len(msgBytes))

	// Set SchemaVersion
	finalMsg[0] = byte(version)

	// Copy serialized message
	copy(finalMsg[HeaderSize:], msgBytes)

	// Calculate and set CRC-32 of the serialized message
	crc := crc32.ChecksumIEEE(finalMsg[HeaderSize:])
	binary.BigEndian.PutUint32(finalMsg[VersionSize:HeaderSize], crc)

	return finalMsg, nil
}

// Deserialize decodes a byte slice into a RawMessage. It verifies the schema version
// and CRC checksum before using the appropriate deserializer to decode the message.
func (v *EnvelopeSerializer) Deserialize(data []byte, msg *RawMessage) error {
	if len(data) < HeaderSize {
		return NewErrBadMessage(ErrMsgTooShort)
	}

	version := SchemaVersion(data[0])
	deserializer, ok := v.serializers[version]
	if !ok {
		return NewErrUnsupportedEncoding(version.String())
	}

	// Verify CRC
	expectedCRC := binary.BigEndian.Uint32(data[VersionSize:HeaderSize])
	actualCRC := crc32.ChecksumIEEE(data[HeaderSize:])
	if actualCRC != expectedCRC {
		return NewErrBadMessage(ErrMsgCRCFailed)
	}

	err := deserializer.Deserialize(data[HeaderSize:], msg)
	if err != nil {
		return NewErrDeserializationFailed(version.String(), err)
	}
	return nil
}

// Compile time checks to ensure that serializers implement the MessageSerDe interface
var _ MessageSerDe = &EnvelopeSerializer{}
