//go:build unit || !integration

package envelope

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
	manager *Registry
}

func (suite *RegistryTestSuite) SetupTest() {
	suite.manager = NewRegistry()
}

func (suite *RegistryTestSuite) TestRegister() {
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

func (suite *RegistryTestSuite) TestGetType() {
	suite.manager.Register("TestPayload", TestPayload{})

	t, err := suite.manager.getType("TestPayload")
	suite.NoError(err)
	suite.Equal("TestPayload", t.Name())

	_, err = suite.manager.getType("UnknownType")
	suite.Error(err)
}

func (suite *RegistryTestSuite) TestGetName() {
	suite.manager.Register("TestPayload", TestPayload{})

	name, err := suite.manager.getName(TestPayload{})
	suite.NoError(err)
	suite.Equal("TestPayload", name)

	_, err = suite.manager.getName(struct{}{})
	suite.Error(err)
}

func (suite *RegistryTestSuite) TestSerializeDeserialize() {
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

func (suite *RegistryTestSuite) TestSerializeUnregisteredPayload() {
	_, err := suite.manager.Serialize(&Message{Metadata: &Metadata{}, Payload: struct{}{}})
	suite.Error(err)
}

func (suite *RegistryTestSuite) TestDeserializeUnknownType() {
	rawMessage := &EncodedMessage{
		Metadata: &Metadata{
			KeyPayloadEncoding: JSONPayloadType,
			KeyMessageType:     "UnknownType",
		},
		Payload: []byte("{}"),
	}
	_, err := suite.manager.Deserialize(rawMessage)
	suite.Error(err)
}

func (suite *RegistryTestSuite) TestUnsupportedEncoding() {
	suite.manager.Register("TestPayload", TestPayload{})

	message := &Message{
		Metadata: &Metadata{KeyPayloadEncoding: "unsupported"},
		Payload:  TestPayload{},
	}
	_, err := suite.manager.Serialize(message)
	suite.Error(err)

	rawMessage := &EncodedMessage{
		Metadata: &Metadata{KeyPayloadEncoding: "unsupported"},
		Payload:  []byte("{}"),
	}
	_, err = suite.manager.Deserialize(rawMessage)
	suite.Error(err)
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
