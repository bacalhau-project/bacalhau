//go:build unit || !integration

package ncl

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

type JSONMessageSerDeTestSuite struct {
	suite.Suite
	serDe *JSONMessageSerDe
}

func (suite *JSONMessageSerDeTestSuite) SetupTest() {
	suite.serDe = &JSONMessageSerDe{}
}

func (suite *JSONMessageSerDeTestSuite) TestSerializeDeserialize() {
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
			rawMsg, err := suite.serDe.Serialize(originalMsg)
			suite.NoError(err)

			// Deserialize
			payloadType := reflect.TypeOf(tc.payload)
			resultMsg, err := suite.serDe.Deserialize(rawMsg, payloadType)
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

func (suite *JSONMessageSerDeTestSuite) TestDeserializeError() {
	rawMsg := &RawMessage{
		Metadata: &Metadata{"key": "value"},
		Payload:  []byte(`{"invalid": "json"`),
	}
	_, err := suite.serDe.Deserialize(rawMsg, reflect.TypeOf(0))
	suite.Error(err)
}

func TestJSONMessageSerDeTestSuite(t *testing.T) {
	suite.Run(t, new(JSONMessageSerDeTestSuite))
}
