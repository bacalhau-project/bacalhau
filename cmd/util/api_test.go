package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

// TestReadTokenFn is a function type for the ReadToken function for testing
var TestReadTokenFn = ReadToken

// cSpell:disable
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

func TestResolveAuthCredentials(t *testing.T) {
	tests := []struct {
		name                string
		apiKey              string
		basicAuthUsername   string
		basicAuthPassword   string
		wantNewAuthFlow     bool
		wantAuthScheme      string
		wantCredentialValue string
		wantErr             bool
		expectedErrMsg      string
	}{
		{
			name:                "No credentials provided",
			apiKey:              "",
			basicAuthUsername:   "",
			basicAuthPassword:   "",
			wantNewAuthFlow:     false,
			wantAuthScheme:      "",
			wantCredentialValue: "",
			wantErr:             false,
		},
		{
			name:                "Valid API key",
			apiKey:              "test-api-key",
			basicAuthUsername:   "",
			basicAuthPassword:   "",
			wantNewAuthFlow:     true,
			wantAuthScheme:      "Bearer",
			wantCredentialValue: "test-api-key",
			wantErr:             false,
		},
		{
			name:                "Valid basic auth credentials",
			apiKey:              "",
			basicAuthUsername:   "user",
			basicAuthPassword:   "pass",
			wantNewAuthFlow:     true,
			wantAuthScheme:      "Basic",
			wantCredentialValue: "dXNlcjpwYXNz", // Base64 encoded "user:pass"
			wantErr:             false,
		},
		{
			name:                "Missing password in basic auth",
			apiKey:              "",
			basicAuthUsername:   "user",
			basicAuthPassword:   "",
			wantNewAuthFlow:     true,
			wantAuthScheme:      "",
			wantCredentialValue: "",
			wantErr:             true,
			expectedErrMsg:      "BACALHAU_API_USERNAME provided but not BACALHAU_API_PASSWORD",
		},
		{
			name:                "Missing username in basic auth",
			apiKey:              "",
			basicAuthUsername:   "",
			basicAuthPassword:   "pass",
			wantNewAuthFlow:     true,
			wantAuthScheme:      "",
			wantCredentialValue: "",
			wantErr:             true,
			expectedErrMsg:      "BACALHAU_API_PASSWORD provided but not BACALHAU_API_USERNAME",
		},
		{
			name:                "Both API key and basic auth provided",
			apiKey:              "test-api-key",
			basicAuthUsername:   "user",
			basicAuthPassword:   "pass",
			wantNewAuthFlow:     true,
			wantAuthScheme:      "",
			wantCredentialValue: "",
			wantErr:             true,
			expectedErrMsg:      "can't use both BACALHAU_API_KEY and BACALHAU_API_USERNAME/BACALHAU_API_PASSWORD simultaneously",
		},
		{
			name:                "Credentials with whitespace",
			apiKey:              "  test-api-key  ",
			basicAuthUsername:   "",
			basicAuthPassword:   "",
			wantNewAuthFlow:     true,
			wantAuthScheme:      "Bearer",
			wantCredentialValue: "test-api-key",
			wantErr:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewAuthFlow, gotAuthScheme, gotCredentialValue, err := resolveAuthCredentials(
				tt.apiKey,
				tt.basicAuthUsername,
				tt.basicAuthPassword,
			)

			assert.Equal(t, tt.wantNewAuthFlow, gotNewAuthFlow, "newAuthFlow mismatch")
			assert.Equal(t, tt.wantAuthScheme, gotAuthScheme, "authScheme mismatch")
			assert.Equal(t, tt.wantCredentialValue, gotCredentialValue, "credentialValue mismatch")

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractAuthCredentialsFromEnvVariables(t *testing.T) {
	// Save original env values to restore later
	originalAPIKey := os.Getenv("BACALHAU_API_KEY")
	originalUsername := os.Getenv("BACALHAU_API_USERNAME")
	originalPassword := os.Getenv("BACALHAU_API_PASSWORD")

	// Cleanup function to restore original env values
	defer func() {
		os.Setenv("BACALHAU_API_KEY", originalAPIKey)
		os.Setenv("BACALHAU_API_USERNAME", originalUsername)
		os.Setenv("BACALHAU_API_PASSWORD", originalPassword)
	}()

	tests := []struct {
		name         string
		envSetup     map[string]string
		expectedKey  string
		expectedUser string
		expectedPass string
	}{
		{
			name: "No environment variables set",
			envSetup: map[string]string{
				"BACALHAU_API_KEY":      "",
				"BACALHAU_API_USERNAME": "",
				"BACALHAU_API_PASSWORD": "",
			},
			expectedKey:  "",
			expectedUser: "",
			expectedPass: "",
		},
		{
			name: "Only API key set",
			envSetup: map[string]string{
				"BACALHAU_API_KEY":      "test-api-key",
				"BACALHAU_API_USERNAME": "",
				"BACALHAU_API_PASSWORD": "",
			},
			expectedKey:  "test-api-key",
			expectedUser: "",
			expectedPass: "",
		},
		{
			name: "Only basic auth credentials set",
			envSetup: map[string]string{
				"BACALHAU_API_KEY":      "",
				"BACALHAU_API_USERNAME": "testuser",
				"BACALHAU_API_PASSWORD": "testpass",
			},
			expectedKey:  "",
			expectedUser: "testuser",
			expectedPass: "testpass",
		},
		{
			name: "All credentials set",
			envSetup: map[string]string{
				"BACALHAU_API_KEY":      "test-api-key",
				"BACALHAU_API_USERNAME": "testuser",
				"BACALHAU_API_PASSWORD": "testpass",
			},
			expectedKey:  "test-api-key",
			expectedUser: "testuser",
			expectedPass: "testpass",
		},
		{
			name: "Credentials with whitespace",
			envSetup: map[string]string{
				"BACALHAU_API_KEY":      "  test-api-key  ",
				"BACALHAU_API_USERNAME": "  testuser  ",
				"BACALHAU_API_PASSWORD": "  testpass  ",
			},
			expectedKey:  "test-api-key",
			expectedUser: "testuser",
			expectedPass: "testpass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for the test
			for key, value := range tt.envSetup {
				os.Setenv(key, value)
			}

			gotKey, gotUser, gotPass := extractAuthCredentialsFromEnvVariables()

			assert.Equal(t, tt.expectedKey, gotKey, "API key mismatch")
			assert.Equal(t, tt.expectedUser, gotUser, "Username mismatch")
			assert.Equal(t, tt.expectedPass, gotPass, "Password mismatch")
		})
	}
}

func TestGenerateAPIRequestsOptions(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "bacalhau-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name          string
		cfg           types.Bacalhau
		profile       *profile.Profile
		baseURL       string
		expectedError string
		verifyOptions func(t *testing.T, opts []clientv2.OptionFn)
	}{
		{
			name: "No profile - defaults to no TLS",
			cfg: types.Bacalhau{
				DataDir: tmpDir,
			},
			profile: nil,
			baseURL: "http://localhost:1234",
			verifyOptions: func(t *testing.T, opts []clientv2.OptionFn) {
				config := clientv2.DefaultConfig()
				for _, opt := range opts {
					opt(&config)
				}
				assert.False(t, config.TLS.UseTLS)
				assert.False(t, config.TLS.Insecure)
				assert.Empty(t, config.TLS.CACert)
				assert.NotNil(t, config.Headers)
			},
		},
		{
			name: "Profile with HTTPS endpoint enables TLS",
			cfg: types.Bacalhau{
				DataDir: tmpDir,
			},
			profile: &profile.Profile{
				Endpoint: "https://example.com:443",
			},
			baseURL: "https://example.com:443",
			verifyOptions: func(t *testing.T, opts []clientv2.OptionFn) {
				config := clientv2.DefaultConfig()
				for _, opt := range opts {
					opt(&config)
				}
				assert.True(t, config.TLS.UseTLS)
				assert.False(t, config.TLS.Insecure)
				assert.NotNil(t, config.Headers)
			},
		},
		{
			name: "Profile with insecure TLS",
			cfg: types.Bacalhau{
				DataDir: tmpDir,
			},
			profile: &profile.Profile{
				Endpoint: "https://example.com:443",
				TLS:      &profile.TLSConfig{Insecure: true},
			},
			baseURL: "https://example.com:443",
			verifyOptions: func(t *testing.T, opts []clientv2.OptionFn) {
				config := clientv2.DefaultConfig()
				for _, opt := range opts {
					opt(&config)
				}
				assert.True(t, config.TLS.UseTLS)
				assert.True(t, config.TLS.Insecure)
				assert.NotNil(t, config.Headers)
			},
		},
		{
			name: "Profile with HTTP endpoint - no TLS",
			cfg: types.Bacalhau{
				DataDir: tmpDir,
			},
			profile: &profile.Profile{
				Endpoint: "http://localhost:1234",
			},
			baseURL: "http://localhost:1234",
			verifyOptions: func(t *testing.T, opts []clientv2.OptionFn) {
				config := clientv2.DefaultConfig()
				for _, opt := range opts {
					opt(&config)
				}
				assert.False(t, config.TLS.UseTLS)
				assert.False(t, config.TLS.Insecure)
				assert.NotNil(t, config.Headers)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &APIClientManager{
				cfg:     tt.cfg,
				baseURL: tt.baseURL,
				profile: tt.profile,
			}

			opts, err := cm.generateAPIRequestsOptions()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, opts)
			assert.Len(t, opts, 4) // TLS, InsecureTLS, CA cert, headers

			if tt.verifyOptions != nil {
				tt.verifyOptions(t, opts)
			}

			// Verify headers are always present
			config := clientv2.DefaultConfig()
			for _, opt := range opts {
				opt(&config)
			}
			assert.NotNil(t, config.Headers)
			assert.Contains(t, config.Headers, apimodels.HTTPHeaderBacalhauGitVersion)
		})
	}
}

func TestAPIClientManagerNoProfile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bacalhau-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create an APIClientManager without a profile (empty baseURL)
	cm := &APIClientManager{
		cfg: types.Bacalhau{
			DataDir: tmpDir,
		},
		baseURL: "", // No profile configured
		profile: nil,
	}

	t.Run("GetUnauthenticatedAPIClient returns error when no profile", func(t *testing.T) {
		client, err := cm.GetUnauthenticatedAPIClient()
		assert.Nil(t, client)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNoProfile)
		assert.Contains(t, err.Error(), "no profile configured")
		assert.Contains(t, err.Error(), "bacalhau profile save")
	})

	t.Run("GetAuthenticatedAPIClient returns error when no profile", func(t *testing.T) {
		client, err := cm.GetAuthenticatedAPIClient()
		assert.Nil(t, client)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNoProfile)
		assert.Contains(t, err.Error(), "no profile configured")
	})
}
