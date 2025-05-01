package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnownCreateStrategies(t *testing.T) {
	expected := []string{Dir.String(), File.String(), NoCreate.String()}
	actual := KnownCreateStrategies()

	assert.Equal(t, expected, actual, "AllowedCreateStrategies should return all valid strategies")
}

func TestCreateStrategyFromString(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expected      CreateStrategy
		expectError   bool
		errorContains string
	}{
		{
			name:        "directory strategy",
			input:       "dir",
			expected:    Dir,
			expectError: false,
		},
		{
			name:        "file strategy",
			input:       "file",
			expected:    File,
			expectError: false,
		},
		{
			name:        "nocreate strategy",
			input:       "nocreate",
			expected:    NoCreate,
			expectError: false,
		},
		{
			name:        "empty string uses default",
			input:       "",
			expected:    DefaultCreateStrategy,
			expectError: false,
		},
		{
			name:          "invalid strategy",
			input:         "invalid",
			expected:      "",
			expectError:   true,
			errorContains: "invalid CreateAs value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategy, err := CreateStrategyFromString(tc.input)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, strategy)
			}
		})
	}
}
func TestCreateStrategyFromStringCaseInsensitive(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected CreateStrategy
	}{
		{
			name:     "dir uppercase",
			input:    "DIR",
			expected: Dir,
		},
		{
			name:     "dir mixed case",
			input:    "DiR",
			expected: Dir,
		},
		{
			name:     "file uppercase",
			input:    "FILE",
			expected: File,
		},
		{
			name:     "file mixed case",
			input:    "FiLe",
			expected: File,
		},
		{
			name:     "nocreate uppercase",
			input:    "NOCREATE",
			expected: NoCreate,
		},
		{
			name:     "nocreate mixed case",
			input:    "NoCreAte",
			expected: NoCreate,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategy, err := CreateStrategyFromString(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, strategy)
		})
	}
}
