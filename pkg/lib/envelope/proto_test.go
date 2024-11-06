//go:build unit || !integration

package envelope

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProtoSerializerTestSuite struct {
	suite.Suite
	serializer *ProtoMessageSerializer
}

func (suite *ProtoSerializerTestSuite) SetupTest() {
	suite.serializer = &ProtoMessageSerializer{}
}

func (suite *ProtoSerializerTestSuite) TestSerializeDeserialize() {
	original := &EncodedMessage{
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
