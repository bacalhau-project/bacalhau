//go:build unit || !integration

package ncl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

type EncoderTestSuite struct {
	suite.Suite
	serializer *envelope.Serializer
	registry   *envelope.Registry
	encoder    *encoder
}

func (suite *EncoderTestSuite) SetupTest() {
	suite.serializer = envelope.NewSerializer()
	suite.registry = envelope.NewRegistry()
	suite.Require().NoError(suite.registry.Register(TestPayloadType, TestPayload{}))

	var err error
	suite.encoder, err = newEncoder(encoderConfig{
		source:            "test-source",
		messageSerializer: suite.serializer,
		messageRegistry:   suite.registry,
	})
	suite.Require().NoError(err)
}

func (suite *EncoderTestSuite) TestEncodeDecodeRoundTrip() {
	// Create test message
	event := TestPayload{Message: "Hello, World!"}
	message := envelope.NewMessage(event)

	// Encode
	data, err := suite.encoder.encode(message)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data)

	// Decode
	decoded, err := suite.encoder.decode(data)
	suite.Require().NoError(err)

	// Verify metadata
	suite.Equal("test-source", decoded.Metadata.Get(KeySource))
	suite.Equal(TestPayloadType, decoded.Metadata.Get(envelope.KeyMessageType))
	suite.True(decoded.Metadata.Has(KeyEventTime))

	// Verify payload
	suite.True(decoded.IsType(TestPayload{}))
	payload, ok := decoded.GetPayload(TestPayload{})
	suite.True(ok, "payload type not matched")
	suite.Equal(event, payload)
}

func (suite *EncoderTestSuite) TestEncodeWithExistingMetadata() {
	// Create message with existing metadata
	event := TestPayload{Message: "With Metadata"}
	message := envelope.NewMessage(event).
		WithMetadataValue("CustomKey", "CustomValue")

	// Encode
	data, err := suite.encoder.encode(message)
	suite.Require().NoError(err)

	// Decode and verify
	decoded, err := suite.encoder.decode(data)
	suite.Require().NoError(err)

	// Original metadata should be preserved
	suite.Equal("CustomValue", decoded.Metadata.Get("CustomKey"))
	// Encoder metadata should be added
	suite.Equal("test-source", decoded.Metadata.Get(KeySource))
	suite.True(decoded.Metadata.Has(KeyEventTime))
}

func (suite *EncoderTestSuite) TestEncodeWithExistingTime() {
	// Create message with existing event time
	event := TestPayload{Message: "With Time"}
	existingTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	message := envelope.NewMessage(event)
	message.Metadata.SetTime(KeyEventTime, existingTime)

	// Encode
	data, err := suite.encoder.encode(message)
	suite.Require().NoError(err)

	// Add debug logging to check intermediate state
	afterEncode := message.Metadata.GetTime(KeyEventTime)
	suite.T().Logf("Time after encode: %v", afterEncode)

	// Decode and verify
	decoded, err := suite.encoder.decode(data)
	suite.Require().NoError(err)

	decodedTime := decoded.Metadata.GetTime(KeyEventTime)
	suite.T().Logf("Time after decode: %v", decodedTime)

	// Use UTC() in comparison to ensure both times are in UTC
	suite.Equal(existingTime.UTC(), decodedTime.UTC(),
		"Times should match when both are in UTC")
}

func (suite *EncoderTestSuite) TestDecodeInvalidData() {
	testCases := []struct {
		name     string
		data     []byte
		errorMsg string
	}{
		{
			name:     "empty data",
			data:     []byte{},
			errorMsg: "failed to deserialize message envelope",
		},
		{
			name:     "invalid json",
			data:     []byte("invalid json"),
			errorMsg: "failed to deserialize message envelope",
		},
		{
			name:     "invalid envelope",
			data:     []byte(`{"invalid": "envelope"}`),
			errorMsg: "failed to deserialize message envelope",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			decoded, err := suite.encoder.decode(tc.data)
			suite.Error(err)
			suite.Contains(err.Error(), tc.errorMsg)
			suite.Nil(decoded)
		})
	}
}

func (suite *EncoderTestSuite) TestEncodeWithUnregisteredType() {
	// Create message with unregistered type
	type UnregisteredPayload struct {
		Data string
	}
	message := envelope.NewMessage(UnregisteredPayload{Data: "test"})

	// Attempt to encode
	data, err := suite.encoder.encode(message)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to serialize into raw message")
	suite.Empty(data)
}

func (suite *EncoderTestSuite) TestErrorResponseRegistration() {
	// Create new registry for this test
	registry := envelope.NewRegistry()
	suite.Require().NoError(registry.Register(TestPayloadType, TestPayload{}))

	// First encoder should register error response type
	encoder1, err := newEncoder(encoderConfig{
		source:            "test-source",
		messageSerializer: suite.serializer,
		messageRegistry:   registry,
	})
	suite.Require().NoError(err)
	suite.NotNil(encoder1)

	// Second encoder using same registry should handle already registered case
	encoder2, err := newEncoder(encoderConfig{
		source:            "test-source-2",
		messageSerializer: suite.serializer,
		messageRegistry:   registry,
	})
	suite.Require().NoError(err)
	suite.NotNil(encoder2)

	// Verify we can encode/decode error responses with both encoders
	errorResp := NewErrorResponse(StatusServerError, "test error")
	data, err := encoder1.encode(errorResp.ToEnvelope())
	suite.Require().NoError(err)

	decoded, err := encoder2.decode(data)
	suite.Require().NoError(err)

	payload, ok := decoded.GetPayload(&ErrorResponse{})
	suite.True(ok)
	errResp := payload.(*ErrorResponse)
	suite.Equal(StatusServerError, errResp.StatusCode)
	suite.Equal("test error", errResp.Message)
}

func TestEncoderTestSuite(t *testing.T) {
	suite.Run(t, new(EncoderTestSuite))
}
