package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// TestReadTokenFn is a function type for the ReadToken function for testing
var TestReadTokenFn = ReadToken

func TestParseURL(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		defaultPort      int
		expectedValidity bool
		expectedURL      string
		expectedScheme   string
	}{
		// Valid URLs - Domains
		{
			name:             "Simple domain without port",
			input:            "http://example.com",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://example.com:80",
			expectedScheme:   "http",
		},
		{
			name:             "Domain with custom port",
			input:            "http://example.com:1234",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://example.com:1234",
			expectedScheme:   "http",
		},
		{
			name:             "HTTPS domain with default port",
			input:            "https://example.com",
			defaultPort:      443,
			expectedValidity: true,
			expectedURL:      "https://example.com:443",
			expectedScheme:   "https",
		},

		// Valid URLs - IPv4
		{
			name:             "IPv4 without port",
			input:            "http://192.168.1.1",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://192.168.1.1:80",
			expectedScheme:   "http",
		},
		{
			name:             "IPv4 with custom port",
			input:            "https://192.168.1.1:8443",
			defaultPort:      443,
			expectedValidity: true,
			expectedURL:      "https://192.168.1.1:8443",
			expectedScheme:   "https",
		},

		// Valid URLs - IPv6
		{
			name:             "IPv6 without port - with brackets",
			input:            "http://[2001:db8::1]",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://[2001:db8::1]:80",
			expectedScheme:   "http",
		},
		{
			name:             "IPv6 with custom port - with brackets",
			input:            "https://[2001:db8::1]:8443",
			defaultPort:      443,
			expectedValidity: true,
			expectedURL:      "https://[2001:db8::1]:8443",
			expectedScheme:   "https",
		},
		{
			name:             "IPv6 without port - without brackets",
			input:            "http://2001:db8::1",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://[2001:db8::1]:80",
			expectedScheme:   "http",
		},

		// Invalid URLs
		{
			name:             "Empty string",
			input:            "",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},
		{
			name:             "Invalid scheme",
			input:            "ftp://example.com",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},
		{
			name:             "Missing scheme",
			input:            "example.com",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},
		{
			name:             "URL with path",
			input:            "http://example.com/path",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},
		{
			name:             "URL with query parameters",
			input:            "http://example.com?query=1",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},
		{
			name:             "URL with fragment",
			input:            "http://example.com#fragment",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},
		{
			name:             "Invalid port number format",
			input:            "http://example.com:abc",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},

		// Invalid URLs - IPv4
		{
			name:             "IPv4 bare",
			input:            "192.168.1.1",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},

		{
			name:             "IPv4 bare with port",
			input:            "192.168.1.1:1234",
			defaultPort:      80,
			expectedValidity: false,
			expectedURL:      "",
			expectedScheme:   "",
		},

		// Edge cases
		{
			name:             "URL with whitespace",
			input:            "  http://example.com  ",
			defaultPort:      80,
			expectedValidity: true,
			expectedURL:      "http://example.com:80",
			expectedScheme:   "http",
		},
		{
			name:             "localhost",
			input:            "http://localhost",
			defaultPort:      1234,
			expectedValidity: true,
			expectedURL:      "http://localhost:1234",
			expectedScheme:   "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid, gotURL, gotScheme := parseURL(tt.input, tt.defaultPort)
			assert.Equal(t, tt.expectedValidity, gotValid, "parseURL() validity check")
			assert.Equal(t, tt.expectedURL, gotURL, "parseURL() URL check")
			assert.Equal(t, tt.expectedScheme, gotScheme, "parseURL() Scheme check")
		})
	}
}

func TestConstructAPIEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		apiCfg         types.API
		expected       string
		expectedScheme string
	}{
		{
			name: "Basic host and port without TLS",
			apiCfg: types.API{
				Host: "example.com",
				Port: 8080,
				TLS: types.TLS{
					UseTLS: false,
				},
			},
			expected:       "http://example.com:8080",
			expectedScheme: "http",
		},
		{
			name: "Basic host and port with TLS",
			apiCfg: types.API{
				Host: "example.com",
				Port: 8080,
				TLS: types.TLS{
					UseTLS: true,
				},
			},
			expected:       "https://example.com:8080",
			expectedScheme: "https",
		},
		{
			name: "0.0.0.0 host should convert to 127.0.0.1",
			apiCfg: types.API{
				Host: "0.0.0.0",
				Port: 1234,
				TLS: types.TLS{
					UseTLS: false,
				},
			},
			expected:       "http://127.0.0.1:1234",
			expectedScheme: "http",
		},
		{
			name: "IPv4 address",
			apiCfg: types.API{
				Host: "192.168.1.1",
				Port: 9090,
				TLS: types.TLS{
					UseTLS: false,
				},
			},
			expected:       "http://192.168.1.1:9090",
			expectedScheme: "http",
		},
		{
			name: "Complete URL as host without TLS",
			apiCfg: types.API{
				Host: "http://api.example.org",
				Port: 9999, // Should be ignored
				TLS: types.TLS{
					UseTLS: false, // Should be ignored
				},
			},
			expected:       "http://api.example.org:9999",
			expectedScheme: "http",
		},
		{
			name: "Complete URL as host with port",
			apiCfg: types.API{
				Host: "https://api.example.org:8443",
				Port: 9999, // Should be ignored in favor of the URL's port
				TLS: types.TLS{
					UseTLS: true, // Should be ignored
				},
			},
			expected:       "https://api.example.org:8443",
			expectedScheme: "https",
		},
		{
			name: "Localhost",
			apiCfg: types.API{
				Host: "localhost",
				Port: 3000,
				TLS: types.TLS{
					UseTLS: false,
				},
			},
			expected:       "http://localhost:3000",
			expectedScheme: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urlResult, detectedScheme := ConstructAPIEndpoint(tt.apiCfg)
			assert.Equal(t, tt.expected, urlResult, "ConstructAPIEndpoint() urlResult")
			assert.Equal(t, tt.expectedScheme, detectedScheme, "ConstructAPIEndpoint() detectedScheme")
		})
	}
}
