//go:build unit || !integration

package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type EventTestSuite struct {
	suite.Suite
	topic models.EventTopic
}

func (suite *EventTestSuite) SetupTest() {
	suite.topic = models.EventTopic("TestTopic")
}

func (suite *EventTestSuite) TestNewEvent() {
	event := models.NewEvent(suite.topic)

	suite.Equal(suite.topic, event.Topic)
	suite.WithinDuration(time.Now(), event.Timestamp, time.Second)
	suite.Empty(event.Details)
}

func (suite *EventTestSuite) TestEventWithMessage() {
	message := "TestMessage"
	event := models.NewEvent(suite.topic).WithMessage(message)

	suite.Equal(message, event.Message)
}

func (suite *EventTestSuite) TestEventWithError() {
	err := fmt.Errorf("TestError")
	event := models.NewEvent(suite.topic).WithError(err)

	suite.Equal(err.Error(), event.Message)
	suite.Equal("true", event.Details[models.DetailsKeyIsError])
}

func (suite *EventTestSuite) TestEventWithHint() {
	hint := "TestHint"
	event := models.NewEvent(suite.topic).WithHint(hint)

	suite.Equal(hint, event.Details[models.DetailsKeyHint])
}

func (suite *EventTestSuite) TestEventWithRetryable() {
	event := models.NewEvent(suite.topic).WithRetryable(true)

	suite.Equal("true", event.Details[models.DetailsKeyRetryable])
}

func (suite *EventTestSuite) TestEventWithFailsExecution() {
	event := models.NewEvent(suite.topic).WithFailsExecution(true)

	suite.Equal("true", event.Details[models.DetailsKeyFailsExecution])
}

func (suite *EventTestSuite) TestEventWithDetails() {
	details := map[string]string{"key1": "value1", "key2": "value2"}
	event := models.NewEvent(suite.topic).WithDetails(details)

	suite.Equal(details, event.Details)
}

func (suite *EventTestSuite) TestEventWithDetail() {
	key := "TestKey"
	value := "TestValue"
	event := models.NewEvent(suite.topic).WithDetail(key, value)

	suite.Equal(value, event.Details[key])
}

func (suite *EventTestSuite) TestEventFromError() {
	err := bacerrors.New("TestError").
		WithHint("TestHint").
		WithRetryable().
		WithFailsExecution().
		WithDetails(map[string]string{"key1": "value1", "key2": "value2"})
	event := models.EventFromError(suite.topic, err)

	suite.Equal("TestError", event.Message)
	suite.Equal(suite.topic, event.Topic)
	suite.Equal("true", event.Details[models.DetailsKeyIsError])
	suite.Equal("true", event.Details[models.DetailsKeyRetryable])
	suite.Equal("true", event.Details[models.DetailsKeyFailsExecution])
	suite.Equal(err.Hint(), event.Details[models.DetailsKeyHint])
	suite.Equal("value1", event.Details["key1"])
	suite.Equal("value2", event.Details["key2"])
}

func (suite *EventTestSuite) TestEventFromErrorNoDetails() {
	err := bacerrors.New("TestError")
	event := models.EventFromError(suite.topic, err)

	suite.Equal("TestError", event.Message)
	suite.Equal(suite.topic, event.Topic)
	suite.Equal("true", event.Details[models.DetailsKeyIsError])
	suite.Contains(event.Details, models.DetailsKeyErrorCode)
	suite.Len(event.Details, 2)
}

func (suite *EventTestSuite) TestEventFromSimpleError() {
	err := fmt.Errorf("TestError")
	event := models.EventFromError(suite.topic, err)

	suite.Equal("TestError", event.Message)
	suite.Equal(suite.topic, event.Topic)
	suite.Equal("true", event.Details[models.DetailsKeyIsError])
	suite.Len(event.Details, 1)
}

func (suite *EventTestSuite) TestHasError() {
	// Test case for an event with an error
	eventWithError := models.NewEvent(suite.topic).WithError(fmt.Errorf("Test error"))
	suite.True(eventWithError.HasError())

	// Test case for an event without an error
	eventWithoutError := models.NewEvent(suite.topic)
	suite.False(eventWithoutError.HasError())
}

func (suite *EventTestSuite) TestHasStateUpdate() {
	// Test case for an event with a state update
	eventWithStateUpdate := models.NewEvent(suite.topic).WithDetail(models.DetailsKeyNewState, "Running")
	suite.True(eventWithStateUpdate.HasStateUpdate())

	// Test case for an event without a state update
	eventWithoutStateUpdate := models.NewEvent(suite.topic)
	suite.False(eventWithoutStateUpdate.HasStateUpdate())
}

func (suite *EventTestSuite) TestGetJobStateIfPresent() {
	// Test case for an event with a valid state update
	validState := models.JobStateTypeRunning
	eventWithValidState := models.NewEvent(suite.topic).WithDetail(models.DetailsKeyNewState, validState.String())
	state, err := eventWithValidState.GetJobStateIfPresent()
	suite.NoError(err)
	suite.Equal(validState, state)

	// Test case for an event without a state update
	eventWithoutState := models.NewEvent(suite.topic)
	state, err = eventWithoutState.GetJobStateIfPresent()
	suite.NoError(err)
	suite.Equal(models.JobStateTypeUndefined, state)

	// Test case for an event with an invalid state update
	invalidState := "InvalidState"
	eventWithInvalidState := models.NewEvent(suite.topic).WithDetail(models.DetailsKeyNewState, invalidState)
	state, err = eventWithInvalidState.GetJobStateIfPresent()
	suite.NoError(err) // models.JobStateType.UnmarshalText() does not return an error for invalid states
	suite.Equal(models.JobStateTypeUndefined, state)
}

func TestEventTestSuite(t *testing.T) {
	suite.Run(t, new(EventTestSuite))
}
