//go:build unit || !integration

package ncl

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type JsonMessageSerializerTestSuite struct {
	suite.Suite
	serializer *JSONMessageSerializer
}

func (suite *JsonMessageSerializerTestSuite) SetupTest() {
	suite.serializer = &JSONMessageSerializer{}
}

func (suite *JsonMessageSerializerTestSuite) TestSerializeDeserialize() {
	original := &RawMessage{
		Metadata: &Metadata{"key": "value"},
		Payload:  []byte(`{"test": "data"}`),
	}

	// Serialize
	data, err := suite.serializer.Serialize(original)
	suite.NoError(err)

	// Deserialize
	result := &RawMessage{}
	err = suite.serializer.Deserialize(data, result)
	suite.NoError(err)

	// Compare
	suite.Equal(original.Metadata, result.Metadata)
	suite.JSONEq(string(original.Payload), string(result.Payload))
}

func (suite *JsonMessageSerializerTestSuite) TestSerializeNilMessage() {
	_, err := suite.serializer.Serialize(nil)
	suite.EqualError(err, ErrNilMessage)
}

func (suite *JsonMessageSerializerTestSuite) TestDeserializeEmptyData() {
	err := suite.serializer.Deserialize([]byte{}, &RawMessage{})
	suite.EqualError(err, ErrEmptyData)
}

func (suite *JsonMessageSerializerTestSuite) TestDeserializeInvalidData() {
	err := suite.serializer.Deserialize([]byte(`invalid json`), &RawMessage{})
	suite.Error(err)
}

func TestJsonMessageSerializerTestSuite(t *testing.T) {
	suite.Run(t, new(JsonMessageSerializerTestSuite))
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
	result := &RawMessage{}
	err = suite.serializer.Deserialize(data, result)
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
	err := suite.serializer.Deserialize([]byte{}, &RawMessage{})
	suite.EqualError(err, ErrEmptyData)
}

func (suite *ProtoSerializerTestSuite) TestDeserializeInvalidData() {
	err := suite.serializer.Deserialize([]byte(`invalid proto`), &RawMessage{})
	suite.Error(err)
}

func TestProtoSerializerTestSuite(t *testing.T) {
	suite.Run(t, new(ProtoSerializerTestSuite))
}
