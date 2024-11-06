//go:build unit || !integration

package envelope

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

type JSONPayloadSerializerTestSuite struct {
	suite.Suite
	serializer *JSONPayloadSerializer
}

func (suite *JSONPayloadSerializerTestSuite) SetupTest() {
	suite.serializer = &JSONPayloadSerializer{}
}

func (suite *JSONPayloadSerializerTestSuite) TestSerializeDeserialize() {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	testCases := []struct {
		name    string
		payload interface{}
	}{
		{"string", "test"},
		{"int", 42},
		{"float", 3.14},
		{"bool", true},
		{"struct", testStruct{Name: "John", Age: 30}},
		{"slice", []int{1, 2, 3}},
		{"map", map[string]int{"one": 1, "two": 2}},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create original message
			originalMsg := &Message{
				Metadata: &Metadata{"key": "value"},
				Payload:  tc.payload,
			}

			// Serialize
			rawMsg, err := suite.serializer.Serialize(originalMsg)
			suite.NoError(err)

			// Deserialize
			payloadType := reflect.TypeOf(tc.payload)
			resultMsg, err := suite.serializer.Deserialize(rawMsg, payloadType)
			suite.NoError(err)

			// Compare original and deserialized metadata
			suite.Equal(originalMsg.Metadata, resultMsg.Metadata)

			// Compare the original and deserialized payloads
			resultPayload, ok := resultMsg.GetPayload(tc.payload)
			suite.True(ok, "payload type not matched")
			suite.Equal(tc.payload, resultPayload)
		})
	}
}

func (suite *JSONPayloadSerializerTestSuite) TestDeserializeError() {
	rawMsg := &EncodedMessage{
		Metadata: &Metadata{"key": "value"},
		Payload:  []byte(`{"invalid": "json"`),
	}
	_, err := suite.serializer.Deserialize(rawMsg, reflect.TypeOf(0))
	suite.Error(err)
}

func TestJSONPayloadSerializerTestSuite(t *testing.T) {
	suite.Run(t, new(JSONPayloadSerializerTestSuite))
}

type JSONRawMessageSerializerTestSuite struct {
	suite.Suite
	serializer *JSONMessageSerializer
}

func (suite *JSONRawMessageSerializerTestSuite) SetupTest() {
	suite.serializer = &JSONMessageSerializer{}
}

func (suite *JSONRawMessageSerializerTestSuite) TestSerializeDeserialize() {
	original := &EncodedMessage{
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
