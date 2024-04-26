//go:build unit || !integration

package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type EventTestSuite struct {
	suite.Suite
	topic EventTopic
}

func (suite *EventTestSuite) SetupTest() {
	suite.topic = EventTopic("TestTopic")
}

func (suite *EventTestSuite) TestNewEvent() {
	event := NewEvent(suite.topic)

	suite.Equal(suite.topic, event.Topic)
	suite.WithinDuration(time.Now(), event.Timestamp, time.Second)
	suite.Empty(event.Details)
}

func (suite *EventTestSuite) TestEventWithMessage() {
	message := "TestMessage"
	event := NewEvent(suite.topic).WithMessage(message)

	suite.Equal(message, event.Message)
}

func (suite *EventTestSuite) TestEventWithError() {
	err := fmt.Errorf("TestError")
	event := NewEvent(suite.topic).WithError(err)

	suite.Equal(err.Error(), event.Message)
	suite.Equal("true", event.Details[DetailsKeyIsError])
}

func (suite *EventTestSuite) TestEventWithHint() {
	hint := "TestHint"
	event := NewEvent(suite.topic).WithHint(hint)

	suite.Equal(hint, event.Details[DetailsKeyHint])
}

func (suite *EventTestSuite) TestEventWithRetryable() {
	event := NewEvent(suite.topic).WithRetryable(true)

	suite.Equal("true", event.Details[DetailsKeyRetryable])
}

func (suite *EventTestSuite) TestEventWithFailsExecution() {
	event := NewEvent(suite.topic).WithFailsExecution(true)

	suite.Equal("true", event.Details[DetailsKeyFailsExecution])
}

func (suite *EventTestSuite) TestEventWithDetails() {
	details := map[string]string{"key1": "value1", "key2": "value2"}
	event := NewEvent(suite.topic).WithDetails(details)

	suite.Equal(details, event.Details)
}

func (suite *EventTestSuite) TestEventWithDetail() {
	key := "TestKey"
	value := "TestValue"
	event := NewEvent(suite.topic).WithDetail(key, value)

	suite.Equal(value, event.Details[key])
}

func (suite *EventTestSuite) TestEventFromError() {
	errMessage := "TestError"
	err := &BaseError{
		message:        errMessage,
		hint:           "TestHint",
		retryable:      true,
		failsExecution: true,
		details:        map[string]string{"key1": "value1", "key2": "value2"},
	}
	event := EventFromError(suite.topic, err)

	suite.Equal(errMessage, event.Message)
	suite.Equal(suite.topic, event.Topic)
	suite.Equal("true", event.Details[DetailsKeyIsError])
	suite.Equal("true", event.Details[DetailsKeyRetryable])
	suite.Equal("true", event.Details[DetailsKeyFailsExecution])
	suite.Equal(err.hint, event.Details[DetailsKeyHint])
	suite.Equal("value1", event.Details["key1"])
	suite.Equal("value2", event.Details["key2"])
}

func (suite *EventTestSuite) TestEventFromErrorNoDetails() {
	errMessage := "TestError"
	err := NewBaseError(errMessage)
	event := EventFromError(suite.topic, err)

	suite.Equal(errMessage, event.Message)
	suite.Equal(suite.topic, event.Topic)
	suite.Equal("true", event.Details[DetailsKeyIsError])
	suite.Len(event.Details, 1)
}

func (suite *EventTestSuite) TestEventFromSimpleError() {
	errMessage := "TestError"
	err := fmt.Errorf(errMessage)
	event := EventFromError(suite.topic, err)

	suite.Equal(errMessage, event.Message)
	suite.Equal(suite.topic, event.Topic)
	suite.Equal("true", event.Details[DetailsKeyIsError])
	suite.Len(event.Details, 1)
}

func TestEventTestSuite(t *testing.T) {
	suite.Run(t, new(EventTestSuite))
}
