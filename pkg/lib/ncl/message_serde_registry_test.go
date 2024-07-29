//go:build unit || !integration

package ncl

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MessageSerDeRegistryTestSuite struct {
	suite.Suite
	manager *MessageSerDeRegistry
}

func (suite *MessageSerDeRegistryTestSuite) SetupTest() {
	suite.manager = NewMessageSerDeRegistry()
}

func (suite *MessageSerDeRegistryTestSuite) TestRegister() {
	err := suite.manager.Register("TestPayload", TestPayload{})
	suite.NoError(err)

	// Test registering the same name twice
	err = suite.manager.Register("TestPayload", TestPayload{})
	suite.Error(err)

	// Test registering with empty name
	err = suite.manager.Register("", TestPayload{})
	suite.Error(err)

	// Test registering nil payload
	err = suite.manager.Register("NilPayload", nil)
	suite.Error(err)
}

func (suite *MessageSerDeRegistryTestSuite) TestGetType() {
	suite.manager.Register("TestPayload", TestPayload{})

	t, err := suite.manager.getType("TestPayload")
	suite.NoError(err)
	suite.Equal("TestPayload", t.Name())

	_, err = suite.manager.getType("UnknownType")
	suite.Error(err)
}

func (suite *MessageSerDeRegistryTestSuite) TestGetName() {
	suite.manager.Register("TestPayload", TestPayload{})

	name, err := suite.manager.getName(TestPayload{})
	suite.NoError(err)
	suite.Equal("TestPayload", name)

	_, err = suite.manager.getName(struct{}{})
	suite.Error(err)
}

func (suite *MessageSerDeRegistryTestSuite) TestSerializeDeserialize() {
	suite.manager.Register("TestPayload", TestPayload{})

	original := &Message{
		Metadata: &Metadata{},
		Payload:  TestPayload{Message: "Test", Value: 42},
	}

	// Serialize
	rawMessage, err := suite.manager.Serialize(original)
	suite.NoError(err)
	suite.Equal(DefaultPayloadEncoding, rawMessage.Metadata.Get(KeyPayloadEncoding))
	suite.Equal("TestPayload", rawMessage.Metadata.Get(KeyMessageType))

	// Deserialize
	deserialized, err := suite.manager.Deserialize(rawMessage)
	suite.NoError(err)

	// Compare
	suite.Equal(original.Metadata, deserialized.Metadata)
	suite.IsType(&TestPayload{}, deserialized.Payload)

	deserializedPayload, ok := deserialized.GetPayload(TestPayload{})
	suite.True(ok, "payload type not matched")
	suite.Equal(original.Payload, deserializedPayload)
}

func (suite *MessageSerDeRegistryTestSuite) TestSerializeUnregisteredPayload() {
	_, err := suite.manager.Serialize(&Message{Metadata: &Metadata{}, Payload: struct{}{}})
	suite.Error(err)
}

func (suite *MessageSerDeRegistryTestSuite) TestDeserializeUnknownType() {
	rawMessage := &RawMessage{
		Metadata: &Metadata{
			KeyPayloadEncoding: JSONPayloadSerDeType,
			KeyMessageType:     "UnknownType",
		},
		Payload: []byte("{}"),
	}
	_, err := suite.manager.Deserialize(rawMessage)
	suite.Error(err)
}

func (suite *MessageSerDeRegistryTestSuite) TestUnsupportedEncoding() {
	suite.manager.Register("TestPayload", TestPayload{})

	message := &Message{
		Metadata: &Metadata{KeyPayloadEncoding: "unsupported"},
		Payload:  TestPayload{},
	}
	_, err := suite.manager.Serialize(message)
	suite.Error(err)

	rawMessage := &RawMessage{
		Metadata: &Metadata{KeyPayloadEncoding: "unsupported"},
		Payload:  []byte("{}"),
	}
	_, err = suite.manager.Deserialize(rawMessage)
	suite.Error(err)
}

func TestMessageSerDeRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(MessageSerDeRegistryTestSuite))
}
