//go:build unit || !integration

package s3

import "testing"

// TestIsAWSEndpoint tests the IsAWSEndpoint function with various inputs.
func TestIsAWSEndpoint(t *testing.T) {
	tests := []struct {
		endpoint string
		want     bool
	}{
		{"s3.us-west-2.amazonaws.com", true},
		{"s3.eu-central-1.amazonaws.com", true},
		{"https://storage.googleapis.com", false},
		{"my-custom-s3.com", false},
		{"localhost:9000", false},
		{"", true}, // An empty endpoint is considered AWS
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			if got := IsAWSEndpoint(tt.endpoint); got != tt.want {
				t.Errorf("IsAWSEndpoint(%q) = %v, want %v", tt.endpoint, got, tt.want)
			}
		})
	}
}
