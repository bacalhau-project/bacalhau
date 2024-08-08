//go:build unit || !integration

package watcher

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type JSONSerializerTestSuite struct {
	suite.Suite
	serializer *JSONSerializer
}

func (suite *JSONSerializerTestSuite) SetupTest() {
	suite.serializer = NewJSONSerializer()
	err := suite.serializer.RegisterType("TestObject", reflect.TypeOf(TestObject{}))
	suite.Require().NoError(err)
}

func (suite *JSONSerializerTestSuite) TestSerializeDeserialize() {
	originalEvent := Event{
		SeqNum:     1,
		Operation:  OperationCreate,
		ObjectType: "TestObject",
		Object: TestObject{
			Name:  "Test",
			Value: 42,
		},
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
	}

	// Serialize
	serialized, err := suite.serializer.Marshal(originalEvent)
	suite.Require().NoError(err)
	suite.NotEmpty(serialized)

	// Deserialize
	var deserializedEvent Event
	err = suite.serializer.Unmarshal(serialized, &deserializedEvent)
	suite.Require().NoError(err)

	// Compare
	suite.Equal(originalEvent.SeqNum, deserializedEvent.SeqNum)
	suite.Equal(originalEvent.Operation, deserializedEvent.Operation)
	suite.Equal(originalEvent.ObjectType, deserializedEvent.ObjectType)
	suite.Equal(originalEvent.Timestamp, deserializedEvent.Timestamp)
	suite.True(deserializedEvent.Timestamp.Equal(originalEvent.Timestamp), "Timestamps should be equal")
	suite.Equal(originalEvent.Timestamp.Location(), deserializedEvent.Timestamp.Location(), "Both timestamps should be in UTC")

	// Compare Object field
	originalObj, ok := originalEvent.Object.(TestObject)
	suite.Require().True(ok)
	deserializedObj, ok := deserializedEvent.Object.(TestObject)
	suite.Require().True(ok)

	suite.Equal(originalObj.Name, deserializedObj.Name)
	suite.Equal(originalObj.Value, deserializedObj.Value)
}

func (suite *JSONSerializerTestSuite) TestSerializeDeserializeNilObject() {
	originalEvent := Event{
		SeqNum:     2,
		Operation:  OperationDelete,
		ObjectType: "TestObject",
		Object:     nil,
		Timestamp:  time.Now().UTC().Truncate(time.Millisecond),
	}

	// Serialize
	serialized, err := suite.serializer.Marshal(originalEvent)
	suite.Require().NoError(err)
	suite.NotEmpty(serialized)

	// Deserialize
	var deserializedEvent Event
	err = suite.serializer.Unmarshal(serialized, &deserializedEvent)
	suite.Require().NoError(err)

	// Compare
	suite.Equal(originalEvent.SeqNum, deserializedEvent.SeqNum)
	suite.Equal(originalEvent.Operation, deserializedEvent.Operation)
	suite.Equal(originalEvent.ObjectType, deserializedEvent.ObjectType)
	suite.Equal(originalEvent.Timestamp, deserializedEvent.Timestamp)
	suite.Nil(deserializedEvent.Object)
}

func (suite *JSONSerializerTestSuite) TestRegisterDuplicateType() {
	err := suite.serializer.RegisterType("TestObject", reflect.TypeOf(TestObject{}))
	suite.Error(err)
	suite.Contains(err.Error(), "already registered")
}

func (suite *JSONSerializerTestSuite) TestRegisterNilType() {
	err := suite.serializer.RegisterType("NilType", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "cannot register nil type")
}

func (suite *JSONSerializerTestSuite) TestRegisterPointerType() {
	err := suite.serializer.RegisterType("PointerType", reflect.TypeOf(&TestObject{}))
	suite.Error(err)
	suite.Contains(err.Error(), "cannot register pointer type")
}

func TestJSONSerializerTestSuite(t *testing.T) {
	suite.Run(t, new(JSONSerializerTestSuite))
}
