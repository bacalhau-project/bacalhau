//go:build unit || !integration

package bacerrors

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorJSONMarshalling(t *testing.T) {
	// Create an error with all fields populated
	originalErr := &errorImpl{
		cause:          "test error",
		hint:           "try this instead",
		retryable:      true,
		failsExecution: true,
		component:      "TestComponent",
		httpStatusCode: 404,
		details: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		code: NotFoundError,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalErr)
	require.NoError(t, err, "Failed to marshal error to JSON")

	// Unmarshal back to a new error
	var unmarshaled errorImpl
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to error")

	// Verify all fields match
	assert.Equal(t, originalErr.cause, unmarshaled.cause, "Cause field mismatch")
	assert.Equal(t, originalErr.hint, unmarshaled.hint, "Hint field mismatch")
	assert.Equal(t, originalErr.retryable, unmarshaled.retryable, "Retryable field mismatch")
	assert.Equal(t, originalErr.failsExecution, unmarshaled.failsExecution, "FailsExecution field mismatch")
	assert.Equal(t, originalErr.component, unmarshaled.component, "Component field mismatch")
	assert.Equal(t, originalErr.httpStatusCode, unmarshaled.httpStatusCode, "HTTPStatusCode field mismatch")
	assert.Equal(t, originalErr.code, unmarshaled.code, "Code field mismatch")
	assert.Equal(t, originalErr.details, unmarshaled.details, "Details field mismatch")
}

func TestErrorJSONMarshallingEmpty(t *testing.T) {
	// Test with minimal fields
	originalErr := &errorImpl{
		cause: "minimal error",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalErr)
	require.NoError(t, err, "Failed to marshal minimal error to JSON")

	// Unmarshal back to a new error
	var unmarshaled errorImpl
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err, "Failed to unmarshal JSON to minimal error")

	// Verify fields
	assert.Equal(t, originalErr.cause, unmarshaled.cause, "Cause field mismatch")
	assert.Empty(t, unmarshaled.hint, "Hint should be empty")
	assert.False(t, unmarshaled.retryable, "Retryable should be false")
	assert.False(t, unmarshaled.failsExecution, "FailsExecution should be false")
	assert.Empty(t, unmarshaled.component, "Component should be empty")
	assert.Zero(t, unmarshaled.httpStatusCode, "HTTPStatusCode should be zero")
	assert.Nil(t, unmarshaled.details, "Details should be nil")
	assert.Zero(t, unmarshaled.code, "Code should be zero value")
}

func TestErrorJSONMarshallingInvalid(t *testing.T) {
	// Test unmarshalling invalid JSON
	invalidJSON := []byte(`{"Cause": "test", "Retryable": "invalid"}`)
	var unmarshaled errorImpl
	err := json.Unmarshal(invalidJSON, &unmarshaled)
	assert.Error(t, err, "Should fail to unmarshal invalid JSON")
}

func TestErrorJSONFieldVisibility(t *testing.T) {
	originalErr := &errorImpl{
		cause:          "test error",
		hint:           "test hint",
		retryable:      true,
		failsExecution: true,
		component:      "TestComponent",
		httpStatusCode: 404,
		details: map[string]string{
			"key": "value",
		},
		code: NotFoundError,
		// These fields should not be marshalled
		wrappedErr:  nil,
		wrappingMsg: "should not appear",
		stack:       nil,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalErr)
	require.NoError(t, err, "Failed to marshal error to JSON")

	// Convert to map to check field presence
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	require.NoError(t, err, "Failed to unmarshal JSON to map")

	// Check that internal fields are not exposed
	assert.NotContains(t, result, "wrappedErr", "wrappedErr should not be in JSON")
	assert.NotContains(t, result, "wrappingMsg", "wrappingMsg should not be in JSON")
	assert.NotContains(t, result, "stack", "stack should not be in JSON")
}
