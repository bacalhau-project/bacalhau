//go:build unit || !integration

package boltjobstore

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestCompositeKeyEncodeDecode(t *testing.T) {
	tests := []struct {
		name        string
		executionID string
		jobID       string
	}{
		{
			name:        "simple IDs",
			executionID: "exec-123",
			jobID:       "job-456",
		},
		{
			name:        "UUID-like IDs",
			executionID: "e-550e8400-e29b-41d4-a716-446655440000",
			jobID:       "j-550e8400-e29b-41d4-a716-446655440001",
		},
		{
			name:        "short IDs",
			executionID: "e1",
			jobID:       "j1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encoding
			compositeKey := encodeExecutionJobKey(tt.executionID, tt.jobID)
			expected := tt.executionID + ":" + tt.jobID
			assert.Equal(t, expected, compositeKey)

			// Test decoding
			decodedExecID, decodedJobID, err := decodeExecutionJobKey(compositeKey)
			assert.NoError(t, err)
			assert.Equal(t, tt.executionID, decodedExecID)
			assert.Equal(t, tt.jobID, decodedJobID)
		})
	}
}

func TestCompositeKeyDecodeErrors(t *testing.T) {
	tests := []struct {
		name         string
		compositeKey string
		expectError  bool
	}{
		{
			name:         "no separator",
			compositeKey: "execjob",
			expectError:  true,
		},
		{
			name:         "multiple separators",
			compositeKey: "exec:job:extra",
			expectError:  true,
		},
		{
			name:         "empty string",
			compositeKey: "",
			expectError:  true,
		},
		{
			name:         "only separator",
			compositeKey: ":",
			expectError:  false, // This should work, just empty parts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeExecutionJobKey(tt.compositeKey)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSplitInProgressIndexKey(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		expectedID   string
		expectedType string
	}{
		{
			name:         "composite key with type",
			key:          "batch:job-123",
			expectedID:   "job-123",
			expectedType: "batch",
		},
		{
			name:         "legacy key without type",
			key:          "job-456",
			expectedID:   "job-456",
			expectedType: "",
		},
		{
			name:         "daemon job",
			key:          "daemon:job-789",
			expectedID:   "job-789",
			expectedType: "daemon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, jobType := splitInProgressIndexKey(tt.key)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedType, jobType)
		})
	}
}

func TestCreateInProgressIndexKey(t *testing.T) {
	job := &models.Job{
		ID:   "job-123",
		Type: "batch",
	}

	key := createInProgressIndexKey(job)
	expected := "batch:job-123"
	assert.Equal(t, expected, key)
}

func TestCreateJobNameIndexKey(t *testing.T) {
	tests := []struct {
		name      string
		jobName   string
		namespace string
		expected  string
	}{
		{
			name:      "simple case",
			jobName:   "my-job",
			namespace: "default",
			expected:  "my-job:default",
		},
		{
			name:      "with special characters",
			jobName:   "job-with-dashes",
			namespace: "my-namespace",
			expected:  "job-with-dashes:my-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := createJobNameIndexKey(tt.jobName, tt.namespace)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestUint64BytesConversion(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{
			name:  "zero value",
			value: 0,
		},
		{
			name:  "small value",
			value: 42,
		},
		{
			name:  "large value",
			value: 18446744073709551615, // max uint64
		},
		{
			name:  "sequence number",
			value: 1234567890,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test round-trip conversion
			bytes := uint64ToBytes(tt.value)
			assert.Len(t, bytes, 8, "uint64 should convert to 8 bytes")

			converted := bytesToUint64(bytes)
			assert.Equal(t, tt.value, converted, "round-trip conversion should preserve value")
		})
	}
}
