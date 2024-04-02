//go:build unit || !integration

package util_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestInterfaceToStringArray(t *testing.T) {
	testcases := []struct {
		name        string
		source      interface{}
		expected    []string
		shouldError bool
	}{
		{
			name:        "nil",
			source:      nil,
			expected:    nil,
			shouldError: false,
		},
		{
			name:        "empty",
			source:      []interface{}{},
			expected:    []string{},
			shouldError: false,
		},
		{
			name:        "string",
			source:      []interface{}{"foo"},
			expected:    []string{"foo"},
			shouldError: false,
		},
		{
			name:        "int",
			source:      []interface{}{1},
			expected:    []string{"1"},
			shouldError: false,
		},
		{
			name:        "float",
			source:      []interface{}{1.1},
			expected:    []string{"1.1"},
			shouldError: false,
		},
		{
			name:        "bool",
			source:      []interface{}{true},
			expected:    []string{"true"},
			shouldError: false,
		},
		{
			name:        "mixed",
			source:      []interface{}{"foo", 1, 1.1, true},
			expected:    []string{"foo", "1", "1.1", "true"},
			shouldError: false,
		},
		{
			name:        "map",
			source:      map[string]interface{}{"foo": "bar"},
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "string array",
			source:      []interface{}{"foo", "bar"},
			expected:    []string{"foo", "bar"},
			shouldError: false,
		},
		{
			name:        "int array",
			source:      []interface{}{1, 2},
			expected:    []string{"1", "2"},
			shouldError: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := util.InterfaceToStringArray(tc.source)
			if tc.shouldError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
