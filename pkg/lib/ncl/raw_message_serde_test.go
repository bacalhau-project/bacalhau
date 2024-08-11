//go:build unit || !integration

package ncl

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type JSONRawMessageSerializerTestSuite struct {
	suite.Suite
	serializer *JSONRawMessageSerializer
}

func (suite *JSONRawMessageSerializerTestSuite) SetupTest() {
	suite.serializer = &JSONRawMessageSerializer{}
}

func (suite *JSONRawMessageSerializerTestSuite) TestSerializeDeserialize() {
	original := &RawMessage{
		Metadata: &Metadata{"key": "value"},
		Payload:  []byte(`{"test": "data"}`),
	}

	// Serialize
	data, err := suite.serializer.Serialize(original)
	suite.NoError(err)

	// Deserialize
	result, err := suite.serializer.Deserialize(data)
	suite.NoError(err)

	// Compare
	suite.Equal(original.Metadata, result.Metadata)
	suite.JSONEq(string(original.Payload), string(result.Payload))
}

func (suite *JSONRawMessageSerializerTestSuite) TestSerializeNilMessage() {
	_, err := suite.serializer.Serialize(nil)
	suite.EqualError(err, ErrNilMessage)
}

func (suite *JSONRawMessageSerializerTestSuite) TestDeserializeEmptyData() {
	_, err := suite.serializer.Deserialize([]byte{})
	suite.EqualError(err, ErrEmptyData)
}

func (suite *JSONRawMessageSerializerTestSuite) TestDeserializeInvalidData() {
	_, err := suite.serializer.Deserialize([]byte(`invalid json`))
	suite.Error(err)
}

func TestJSONRawMessageSerializerTestSuite(t *testing.T) {
	suite.Run(t, new(JSONRawMessageSerializerTestSuite))
}

type ProtoSerializerTestSuite struct {
	suite.Suite
	serializer *ProtoSerializer
}

func (suite *ProtoSerializerTestSuite) SetupTest() {
	suite.serializer = &ProtoSerializer{}
}

func (suite *ProtoSerializerTestSuite) TestSerializeDeserialize() {
	original := &RawMessage{
		Metadata: &Metadata{"key": "value"},
		Payload:  []byte(`test data`),
	}

	// Serialize
	data, err := suite.serializer.Serialize(original)
	suite.NoError(err)

	// Deserialize
	result, err := suite.serializer.Deserialize(data)
	suite.NoError(err)

	// Compare
	suite.Equal(original.Metadata, result.Metadata)
	suite.Equal(original.Payload, result.Payload)
}

func (suite *ProtoSerializerTestSuite) TestSerializeNilMessage() {
	_, err := suite.serializer.Serialize(nil)
	suite.EqualError(err, ErrNilMessage)
}

func (suite *ProtoSerializerTestSuite) TestDeserializeEmptyData() {
	_, err := suite.serializer.Deserialize([]byte{})
	suite.EqualError(err, ErrEmptyData)
}

func (suite *ProtoSerializerTestSuite) TestDeserializeInvalidData() {
	_, err := suite.serializer.Deserialize([]byte(`invalid proto`))
	suite.Error(err)
}

func TestProtoSerializerTestSuite(t *testing.T) {
	suite.Run(t, new(ProtoSerializerTestSuite))
}
