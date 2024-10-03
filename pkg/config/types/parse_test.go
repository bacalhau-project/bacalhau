//go:build unit || !integration

package types_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestCastConfigValueForKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       interface{}
		expected    interface{}
		expectedErr bool
	}{
		{
			name:     "valid string value",
			key:      types.APIHostKey,
			value:    "localhost",
			expected: "localhost",
		},
		{
			name:     "valid int value",
			key:      types.APIPortKey,
			value:    "8080",
			expected: int64(8080),
		},
		{
			name:        "invalid int value",
			key:         types.APIPortKey,
			value:       "not an int",
			expectedErr: true,
		},
		{
			name:     "valid bool value",
			key:      types.WebUIEnabledKey,
			value:    "true",
			expected: true,
		},
		{
			name:        "invalid bool value",
			key:         types.WebUIEnabledKey,
			value:       "not a bool",
			expectedErr: true,
		},
		{
			name:     "valid duration value",
			key:      types.OrchestratorNodeManagerDisconnectTimeoutKey,
			value:    "5m",
			expected: "5m0s",
		},
		{
			name:        "invalid duration value",
			key:         types.OrchestratorNodeManagerDisconnectTimeoutKey,
			value:       "not a duration",
			expectedErr: true,
		},
		{
			name:     "valid single string",
			key:      types.ComputeOrchestratorsKey,
			value:    "nats://127.0.0.1:4222",
			expected: []string{"nats://127.0.0.1:4222"},
		},
		{
			name:     "valid comma-separated string",
			key:      types.ComputeOrchestratorsKey,
			value:    "nats://127.0.0.1:4222,nats://127.0.0.1:4223",
			expected: []string{"nats://127.0.0.1:4222", "nats://127.0.0.1:4223"},
		},
		{
			name:        "invalid string separator ;",
			key:         types.ComputeOrchestratorsKey,
			value:       "nats://127.0.0.1:4222;nats://127.0.0.1:4223",
			expectedErr: true,
		},
		{
			name:        "invalid string separator space",
			key:         types.ComputeOrchestratorsKey,
			value:       "nats://127.0.0.1:4222 nats://127.0.0.1:4223",
			expectedErr: true,
		},
		{
			name:        "mismatched separators and tokens",
			key:         types.ComputeOrchestratorsKey,
			value:       "nats://127.0.0.1:4222,nats://127.0.0.1:4223,",
			expectedErr: true,
		},
		{
			name:     "valid string slice",
			key:      types.ComputeOrchestratorsKey,
			value:    []string{"nats://127.0.0.1:4222", "nats://127.0.0.1:4223"},
			expected: []string{"nats://127.0.0.1:4222", "nats://127.0.0.1:4223"},
		},
		{
			name:     "valid map with single value",
			key:      types.LabelsKey,
			value:    "key1=value1",
			expected: map[string]string{"key1": "value1"},
		},
		{
			name:     "valid map with values",
			key:      types.LabelsKey,
			value:    "key1=value1,key2=value2",
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:        "invalid map value",
			key:         types.LabelsKey,
			value:       "invalid map format",
			expectedErr: true,
		},
		{
			name:        "invalid key",
			key:         "InvalidKey",
			value:       "some value",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := types.CastConfigValueForKey(tt.key, tt.value)

			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCastConfigValueForKey_BoolInput(t *testing.T) {
	result, err := types.CastConfigValueForKey("WebUI.Enabled", true)
	require.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = types.CastConfigValueForKey("WebUI.Enabled", false)
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestCastConfigValueForKey_StringSliceInput(t *testing.T) {
	input := []string{"nats://127.0.0.1:4222", "nats://127.0.0.1:4223"}
	result, err := types.CastConfigValueForKey("Compute.Orchestrators", input)
	require.NoError(t, err)
	assert.Equal(t, input, result)
}

func TestCastConfigValueForKey_UnsupportedType(t *testing.T) {
	_, err := types.CastConfigValueForKey("API.Host", 123) // Passing an int instead of a string or []string
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DEVELOPER ERROR CastConfigValueForKey called with unsupported type: int")
}

func TestAllKeys(t *testing.T) {
	keys := types.AllKeys()

	// Test a few expected keys
	assert.Contains(t, keys, "api.host")
	assert.Equal(t, reflect.TypeOf(""), keys["api.host"])

	assert.Contains(t, keys, "api.port")
	assert.Equal(t, reflect.TypeOf(0), keys["api.port"])

	assert.Contains(t, keys, "webui.enabled")
	assert.Equal(t, reflect.TypeOf(false), keys["webui.enabled"])

	assert.Contains(t, keys, "orchestrator.nodemanager.disconnecttimeout")
	assert.Equal(t, reflect.TypeOf(types.Duration(0)), keys["orchestrator.nodemanager.disconnecttimeout"])

	assert.Contains(t, keys, "compute.orchestrators")
	assert.Equal(t, reflect.TypeOf([]string{}), keys["compute.orchestrators"])

	assert.Contains(t, keys, "labels")
	assert.Equal(t, reflect.TypeOf(map[string]string{}), keys["labels"])
}
