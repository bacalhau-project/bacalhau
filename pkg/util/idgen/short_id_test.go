//go:build unit || !integration

/* spell-checker: disable */

package idgen

import (
	"testing"
)

func TestShortUUID(t *testing.T) {
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
			expectedOutput: "78faf1146a45457e825c40fd2fad768f",
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
			expectedOutput: "12345-abcdef-ghijk",
		},
		{
			name:           "More Than 2 Hyphens",
			input:          "a-b-c-d",
			expectedOutput: "a-b-c-d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortUUID(tt.input)
			if got != tt.expectedOutput {
				t.Errorf("extractPrefix(%q) = %q; want %q", tt.input, got, tt.expectedOutput)
			}
		})
	}
}

func TestShortNodeID(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "UUID Only, No Prefix",
			input:          "78faf114-6a45-457e-825c-40fd2fad768f",
			expectedOutput: "78faf114",
		},
		{
			name:           "Empty String",
			input:          "",
			expectedOutput: "",
		},
		{
			name:           "Invalid UUID Format",
			input:          "12345-abcdef-ghijk",
			expectedOutput: "12345-abcdef-ghijk",
		},
		{
			name:           "More Than 2 Hyphens",
			input:          "a-b-c-d",
			expectedOutput: "a-b-c-d",
		},
		{
			name:           "libp2p node id",
			input:          "QmbkBjSraDGpdG3a4t9hg9k5iFFc6xiJn3Tu1dX17GyMgQ",
			expectedOutput: "QmbkBjSr",
		},
		{
			name:           "hostname",
			input:          "hostname.domain.com",
			expectedOutput: "hostname.domain.com",
		},
		{
			name:           "hostname with subdomain",
			input:          "sub-domain.hostname.domain.com",
			expectedOutput: "sub-domain.hostname.domain.com",
		},
		{
			name:           "aws instance id",
			input:          "i-0a1b2c3d4e5f6g7h8",
			expectedOutput: "i-0a1b2c3d4e5f6g7h8",
		},
		{
			name:           "gcp instance id",
			input:          "4267434651811076444",
			expectedOutput: "4267434651811076444",
		},
		{
			name:           "local hostname",
			input:          "Walid-MacBook.local",
			expectedOutput: "Walid-MacBook.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortNodeID(tt.input)
			if got != tt.expectedOutput {
				t.Errorf("extractPrefix(%q) = %q; want %q", tt.input, got, tt.expectedOutput)
			}
		})
	}
}
