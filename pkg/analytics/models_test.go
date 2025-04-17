package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/log"
)

type ModelsTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

func (s *ModelsTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *ModelsTestSuite) TestEventCreation() {
	// Test basic event creation
	event := NewEvent("test.event", map[string]string{"key": "value"})
	s.Equal("test.event", event.Type)
	s.Equal(map[string]string{"key": "value"}, event.Properties)

	// Test event to log record conversion
	record, err := event.ToLogRecord()
	s.NoError(err)
	s.NotEmpty(record)

	// Walk through attributes and verify they exist
	attributes := make(map[string]string)
	record.WalkAttributes(func(kv log.KeyValue) bool {
		attributes[kv.Key] = kv.Value.AsString()
		return true
	})

	s.Equal("test.event", attributes["event"])
	s.Equal(`{"key":"value"}`, attributes["properties"])
}

func (s *ModelsTestSuite) TestEventToLogRecord() {
	testCases := []struct {
		name          string
		event         *Event
		expectedType  string
		expectedProps string
	}{
		{
			name:          "empty event",
			event:         NewEvent("test.event", nil),
			expectedType:  "test.event",
			expectedProps: "null",
		},
		{
			name:          "simple event",
			event:         NewEvent("test.event", map[string]string{"key": "value"}),
			expectedType:  "test.event",
			expectedProps: `{"key":"value"}`,
		},
		{
			name:          "complex event",
			event:         NewEvent("test.event", map[string]interface{}{"key": 123, "nested": map[string]string{"foo": "bar"}}),
			expectedType:  "test.event",
			expectedProps: `{"key":123,"nested":{"foo":"bar"}}`,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			record, err := tc.event.ToLogRecord()
			s.NoError(err)
			s.NotEmpty(record)

			// Walk through attributes and verify they exist
			attributes := make(map[string]string)
			record.WalkAttributes(func(kv log.KeyValue) bool {
				attributes[kv.Key] = kv.Value.AsString()
				return true
			})

			s.Equal(tc.expectedType, attributes["event"])
			s.Equal(tc.expectedProps, attributes["properties"])
		})
	}
}

func (s *ModelsTestSuite) TestEventTimestamp() {
	event := NewEvent("test.event", map[string]string{"key": "value"})
	record, err := event.ToLogRecord()
	s.NoError(err)
	s.NotEmpty(record)

	// Check that timestamp is recent
	timestamp := record.Timestamp()
	s.True(timestamp.After(time.Now().Add(-time.Second)))
	s.True(timestamp.Before(time.Now().Add(time.Second)))
}
