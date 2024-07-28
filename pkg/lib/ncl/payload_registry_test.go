//go:build unit || !integration

package ncl

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PayloadRegistryTestSuite struct {
	suite.Suite
	registry *PayloadRegistry
}

func (suite *PayloadRegistryTestSuite) SetupTest() {
	suite.registry = NewPayloadRegistry()
}

func (suite *PayloadRegistryTestSuite) TestRegister() {
	err := suite.registry.Register("TestPayload", TestPayload{})
	suite.NoError(err)

	// Test registering the same name twice
	err = suite.registry.Register("TestPayload", TestPayload{})
	suite.Error(err)

	// Test registering with empty name
	err = suite.registry.Register("", TestPayload{})
	suite.Error(err)

	// Test registering nil payload
	err = suite.registry.Register("NilPayload", nil)
	suite.Error(err)
}

func (suite *PayloadRegistryTestSuite) TestGetType() {
	suite.registry.Register("TestPayload", TestPayload{})

	t, err := suite.registry.getType("TestPayload")
	suite.NoError(err)
	suite.Equal("TestPayload", t.Name())

	_, err = suite.registry.getType("UnknownType")
	suite.Error(err)
}

func (suite *PayloadRegistryTestSuite) TestGetName() {
	suite.registry.Register("TestPayload", TestPayload{})

	name, err := suite.registry.getName(TestPayload{})
	suite.NoError(err)
	suite.Equal("TestPayload", name)

	_, err = suite.registry.getName(struct{}{})
	suite.Error(err)
}

func (suite *PayloadRegistryTestSuite) TestSerializeDeserializePayload() {
	suite.registry.Register("TestPayload", TestPayload{})

	original := TestPayload{Message: "Test", Value: 42}
	metadata := &Metadata{}

	// Serialize
	data, err := suite.registry.SerializePayload(metadata, original)
	suite.NoError(err)
	suite.Equal(DefaultPayloadEncoding, metadata.Get(KeyPayloadEncoding))
	suite.Equal("TestPayload", metadata.Get(KeyMessageType))

	// Deserialize
	deserialized, err := suite.registry.DeserializePayload(metadata, data)
	suite.NoError(err)

	// Compare
	suite.IsType(&TestPayload{}, deserialized)
	suite.Equal(original, *deserialized.(*TestPayload))
}

func (suite *PayloadRegistryTestSuite) TestSerializeUnregisteredPayload() {
	_, err := suite.registry.SerializePayload(&Metadata{}, struct{}{})
	suite.Error(err)
}

func (suite *PayloadRegistryTestSuite) TestDeserializeUnknownType() {
	metadata := &Metadata{
		KeyPayloadEncoding: JSONPayloadSerDeType,
		KeyMessageType:     "UnknownType",
	}
	_, err := suite.registry.DeserializePayload(metadata, []byte("{}"))
	suite.Error(err)
}

func (suite *PayloadRegistryTestSuite) TestUnsupportedEncoding() {
	suite.registry.Register("TestPayload", TestPayload{})

	metadata := &Metadata{
		KeyPayloadEncoding: "unsupported",
	}
	_, err := suite.registry.SerializePayload(metadata, TestPayload{})
	suite.Error(err)

	_, err = suite.registry.DeserializePayload(metadata, []byte("{}"))
	suite.Error(err)
}

func TestPayloadRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(PayloadRegistryTestSuite))
}
