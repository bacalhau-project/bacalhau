//go:build unit || !integration

package idgen

import (
	"testing"
)

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "Valid Prefix with UUID",
			input:          "e-78faf114-6a45-457e-825c-40fd2fad768f",
			expectedOutput: "e-78faf114",
		},
		{
			name:           "UUID Only, No Prefix",
			input:          "78faf114-6a45-457e-825c-40fd2fad768f",
			expectedOutput: "78faf114",
		},
		{
			name:           "String with Less Than 2 Hyphens",
			input:          "e-78faf114",
			expectedOutput: "e-78faf114",
		},
		{
			name:           "String with No Hyphens",
			input:          "78faf1146a45457e825c40fd2fad768f",
			expectedOutput: "78faf114",
		},
		{
			name:           "Already short string",
			input:          "78faf",
			expectedOutput: "78faf",
		},
		{
			name:           "Empty String",
			input:          "",
			expectedOutput: "",
		},
		{
			name:           "Invalid UUID Format",
			input:          "12345-abcdef-ghijk",
			expectedOutput: "12345-ab",
		},
		{
			name:           "More Than 2 Hyphens",
			input:          "a-b-c-d",
			expectedOutput: "a-b-c-d",
		},
		{
			name:           "node id",
			input:          "QmbkBjSraDGpdG3a4t9hg9k5iFFc6xiJn3Tu1dX17GyMgQ",
			expectedOutput: "QmbkBjSr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortID(tt.input)
			if got != tt.expectedOutput {
				t.Errorf("extractPrefix(%q) = %q; want %q", tt.input, got, tt.expectedOutput)
			}
		})
	}
}
