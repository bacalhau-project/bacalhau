//go:build unit || !integration

package ncl

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

type JsonPayloadSerDeTestSuite struct {
	suite.Suite
	serDe *JSONPayloadSerDe
}

func (suite *JsonPayloadSerDeTestSuite) SetupTest() {
	suite.serDe = &JSONPayloadSerDe{}
}

func (suite *JsonPayloadSerDeTestSuite) TestSerializeDeserialize() {
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
			// Serialize
			data, err := suite.serDe.SerializePayload(nil, tc.payload)
			suite.NoError(err)

			// Deserialize
			payloadType := reflect.TypeOf(tc.payload)
			result, err := suite.serDe.DeserializePayload(nil, payloadType, data)
			suite.NoError(err)

			// Compare original and deserialized
			suite.Equal(tc.payload, reflect.ValueOf(result).Elem().Interface())
		})
	}
}

func (suite *JsonPayloadSerDeTestSuite) TestDeserializePayloadError() {
	_, err := suite.serDe.DeserializePayload(nil, reflect.TypeOf(0), []byte(`{"invalid": "json"`))
	suite.Error(err)
}

func TestJsonPayloadSerDeTestSuite(t *testing.T) {
	suite.Run(t, new(JsonPayloadSerDeTestSuite))
}
