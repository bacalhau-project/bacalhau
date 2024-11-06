//go:build unit || !integration

package envelope

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type MessageTestSuite struct {
	suite.Suite
}

func (suite *MessageTestSuite) TestNewMetadataFromMap() {
	m := map[string]string{"key": "value"}
	metadata := NewMetadataFromMap(m)
	suite.Equal("value", (*metadata)["key"])

	// Test nil map
	nilMetadata := NewMetadataFromMap(nil)
	suite.NotNil(nilMetadata)
	suite.Empty(*nilMetadata)
}

func (suite *MessageTestSuite) TestNewMetadataFromMapCopy() {
	m := map[string]string{"key": "value"}
	metadata := NewMetadataFromMapCopy(m)
	suite.Equal("value", (*metadata)["key"])

	// Modify original map
	m["key"] = "new value"
	suite.Equal("value", (*metadata)["key"])
}

func (suite *MessageTestSuite) TestMetadataGetSet() {
	m := Metadata{}
	m.Set("key", "value")
	suite.Equal("value", m.Get("key"))
	suite.True(m.Has("key"))
	suite.False(m.Has("nonexistent"))
}

func (suite *MessageTestSuite) TestMetadataSetGetInt() {
	m := Metadata{}
	m.SetInt("int", 42)
	suite.Equal(42, m.GetInt("int"))
	suite.Equal(0, m.GetInt("nonexistent"))
}

func (suite *MessageTestSuite) TestMetadataSetGetInt64() {
	m := Metadata{}
	m.SetInt64("int64", 42)
	suite.Equal(int64(42), m.GetInt64("int64"))
	suite.Equal(int64(0), m.GetInt64("nonexistent"))
}

func (suite *MessageTestSuite) TestMetadataSetGetTime() {
	m := Metadata{}
	now := time.Now()
	m.SetTime("time", now)
	suite.Equal(now.UnixNano(), m.GetTime("time").UnixNano())
	suite.Equal(time.Time{}, m.GetTime("nonexistent"))
}

func (suite *MessageTestSuite) TestMetadataToMap() {
	m := Metadata{"key": "value"}
	suite.Equal(map[string]string{"key": "value"}, m.ToMap())
}

func (suite *MessageTestSuite) TestMessageIsType() {
	msg := &Message{
		Metadata: &Metadata{},
		Payload:  "test payload",
	}
	suite.True(msg.IsType(""))
	suite.False(msg.IsType(42))

	intMsg := &Message{
		Metadata: &Metadata{},
		Payload:  42,
	}
	suite.True(intMsg.IsType(0))
	suite.True(intMsg.IsType(new(int)))
}

func (suite *MessageTestSuite) TestMessageGetPayload() {
	payload := TestPayload{Message: "test payload", Value: 42}
	msg := &Message{
		Metadata: &Metadata{},
		Payload:  payload,
	}

	// Get payload as TestPayload
	retrieved, ok := msg.GetPayload(TestPayload{})
	suite.True(ok)
	suite.IsType(TestPayload{}, retrieved)
	suite.Equal(payload, retrieved)

	// Get payload as pointer to TestPayload
	retrieved, ok = msg.GetPayload(&TestPayload{})
	suite.True(ok)
	suite.IsType((*TestPayload)(nil), retrieved)
	suite.Equal(payload.Message, retrieved.(*TestPayload).Message)

	// Test with incorrect type
	_, ok = msg.GetPayload("")
	suite.False(ok)

	// Test with pointer to struct payload
	ptrPayload := &TestPayload{Message: "pointer payload", Value: 84}
	ptrMsg := &Message{
		Metadata: &Metadata{},
		Payload:  ptrPayload,
	}

	// Get pointer payload as pointer
	retrieved, ok = ptrMsg.GetPayload(&TestPayload{})
	suite.True(ok)
	suite.IsType((*TestPayload)(nil), retrieved)
	suite.Equal(ptrPayload.Message, retrieved.(*TestPayload).Message)

	// Get pointer payload as value
	retrieved, ok = ptrMsg.GetPayload(TestPayload{})
	suite.True(ok)
	suite.IsType(TestPayload{}, retrieved)
	suite.Equal(ptrPayload.Message, retrieved.(TestPayload).Message)

}

func TestMessageTestSuite(t *testing.T) {
	suite.Run(t, new(MessageTestSuite))
}
