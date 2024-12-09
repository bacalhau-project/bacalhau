package util

import "testing"

func TestParseURL(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		defaultPort      int
		expectedValidity bool
		expectedURL      string
	}{
		// Valid URLs - Domains
		{
			name:             "Simple domain without port",
			input:            "http://example.com",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://example.com:80",
		},
		{
			name:             "Domain with custom port",
			input:            "http://example.com:1234",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://example.com:1234",
		},
		{
			name:             "HTTPS domain with default port",
			input:            "https://example.com",
			defaultPort:      443,
			expectedValidity: true,
			expectedURL:      "https://example.com:443",
		},

		// Valid URLs - IPv4
		{
			name:             "IPv4 without port",
			input:            "http://192.168.1.1",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://192.168.1.1:80",
		},
		{
			name:             "IPv4 with custom port",
			input:            "https://192.168.1.1:8443",
			defaultPort:      443,
			expectedValidity: true,
			expectedURL:      "https://192.168.1.1:8443",
		},

		// Valid URLs - IPv6
		{
			name:             "IPv6 without port - with brackets",
			input:            "http://[2001:db8::1]",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://[2001:db8::1]:80",
		},
		{
			name:             "IPv6 with custom port - with brackets",
			input:            "https://[2001:db8::1]:8443",
			defaultPort:      443,
			expectedValidity: true,
			expectedURL:      "https://[2001:db8::1]:8443",
		},
		{
			name:             "IPv6 without port - without brackets",
			input:            "http://2001:db8::1",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://[2001:db8::1]:80",
		},

		// Invalid URLs
		{
			name:             "Empty string",
			input:            "",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},
		{
			name:             "Invalid scheme",
			input:            "ftp://example.com",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},
		{
			name:             "Missing scheme",
			input:            "example.com",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},
		{
			name:             "URL with path",
			input:            "http://example.com/path",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},
		{
			name:             "URL with query parameters",
			input:            "http://example.com?query=1",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},
		{
			name:             "URL with fragment",
			input:            "http://example.com#fragment",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},
		{
			name:             "Invalid port number format",
			input:            "http://example.com:abc",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},

		// Invalid URLs - IPv4
		{
			name:             "IPv4 bare",
			input:            "192.168.1.1",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},

		{
			name:             "IPv4 bare with port",
			input:            "192.168.1.1:1234",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
		},

		// Edge cases
		{
			name:             "URL with whitespace",
			input:            "  http://example.com  ",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://example.com:80",
		},
		{
			name:             "localhost",
			input:            "http://localhost",
			defaultPort:      1234,
			expectedValidity: true,
			expectedURL:      "http://localhost:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid, gotURL := parseURL(tt.input, tt.defaultPort)
			if gotValid != tt.expectedValidity {
				t.Errorf("parseURL() got validity = '%v', expected '%v'", gotValid, tt.expectedValidity)
			}
			if gotURL != tt.expectedURL {
				t.Errorf("parseURL() got url = '%v', expected '%v'", gotURL, tt.expectedURL)
			}
		})
	}
}
