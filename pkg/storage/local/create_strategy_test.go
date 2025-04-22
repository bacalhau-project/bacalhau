package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedCreateStrategies(t *testing.T) {
	expected := []string{Infer.String(), Dir.String(), File.String(), NoCreate.String()}
	actual := AllowedCreateStrategies()

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
			name:        "infer strategy",
			input:       "infer",
			expected:    Infer,
			expectError: false,
		},
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

func TestInferCreateStrategyFromPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected CreateStrategy
	}{
		{
			name:     "empty path should be directory",
			path:     "",
			expected: Dir,
		},
		{
			name:     "path with trailing slash should be directory",
			path:     "/path/to/dir/",
			expected: Dir,
		},
		{
			name:     "path without trailing slash should be file",
			path:     "/path/to/file",
			expected: File,
		},
		{
			name:     "path with extension should be file",
			path:     "/path/to/file.txt",
			expected: File,
		},
		{
			name:     "root directory should be directory",
			path:     "/",
			expected: Dir,
		},
		{
			name:     "relative path to file",
			path:     "file.txt",
			expected: File,
		},
		{
			name:     "relative path to directory with trailing slash",
			path:     "dir/",
			expected: Dir,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			strategy := InferCreateStrategyFromPath(tc.path)
			assert.Equal(t, tc.expected, strategy)
		})
	}
}
