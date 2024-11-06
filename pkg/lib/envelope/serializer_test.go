//go:build unit || !integration

package envelope

import (
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SerializerTestSuite struct {
	suite.Suite
	serializer *Serializer
}

func (suite *SerializerTestSuite) SetupTest() {
	suite.serializer = NewSerializer()
}

func (suite *SerializerTestSuite) TestSerializeDeserialize() {
	for _, schemaVersion := range []SchemaVersion{
		SchemaVersionJSONV1,
		SchemaVersionProtobufV1,
	} {
		suite.Run(schemaVersion.String(), func() {
			original := &EncodedMessage{
				Metadata: &Metadata{"key": "value"},
				Payload:  []byte(`{"test": "data"}`),
			}

			// Set the schema version for this test
			suite.serializer.WithSerializationVersion(schemaVersion)

			// Serialize
			data, err := suite.serializer.Serialize(original)
			suite.NoError(err)

			// Check envelope structure
			suite.Equal(byte(schemaVersion), data[0])

			// Verify CRC
			expectedCRC := binary.BigEndian.Uint32(data[VersionSize:HeaderSize])
			actualCRC := crc32.ChecksumIEEE(data[HeaderSize:])
			suite.Equal(expectedCRC, actualCRC)

			// Deserialize
			result, err := suite.serializer.Deserialize(data)
			suite.NoError(err)

			// Compare
			suite.Equal(original.Metadata, result.Metadata)
			if schemaVersion == SchemaVersionJSONV1 {
				suite.JSONEq(string(original.Payload), string(result.Payload))
			} else {
				suite.Equal(original.Payload, result.Payload)
			}
		})
	}
}

func (suite *SerializerTestSuite) TestDeserializeInvalidVersion() {
	data := []byte{255, 0, 0, 0, 0} // Invalid version
	_, err := suite.serializer.Deserialize(data)
	suite.Error(err)
	suite.IsType(&ErrUnsupportedEncoding{}, err)
}

func (suite *SerializerTestSuite) TestDeserializeInvalidCRC() {
	original := &EncodedMessage{
		Metadata: &Metadata{"key": "value"},
		Payload:  []byte(`{"test": "data"}`),
	}

	data, _ := suite.serializer.Serialize(original)
	// Corrupt CRC
	data[1] ^= 0xFF

	_, err := suite.serializer.Deserialize(data)
	suite.Error(err)
	suite.IsType(&ErrBadMessage{}, err)
	suite.Contains(err.Error(), ErrMsgCRCFailed)
}

func (suite *SerializerTestSuite) TestDeserializeShortMessage() {
	data := []byte{1} // Too short
	_, err := suite.serializer.Deserialize(data)
	suite.Error(err)
	suite.IsType(&ErrBadMessage{}, err)
	suite.Contains(err.Error(), ErrMsgTooShort)
}

func (suite *SerializerTestSuite) TestSerializeNilMessage() {
	_, err := suite.serializer.Serialize(nil)
	suite.Error(err)
	suite.IsType(&ErrSerializationFailed{}, err)
}

func TestSerializerTestSuite(t *testing.T) {
	suite.Run(t, new(SerializerTestSuite))
}
