package sso

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/profile"
	"github.com/bacalhau-project/bacalhau/pkg/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSSOLoginCmd tests the creation of the SSO command
func TestNewSSOLoginCmd(t *testing.T) {
	cmd := NewSSOLoginCmd()

	assert.NotNil(t, cmd, "Command should not be nil")
	assert.Equal(t, "login", cmd.Use, "Command use should be 'sso'")
	assert.Contains(t, cmd.Short, "Login using SSO", "Command should have appropriate short description")
}

// TestPrintDeviceCodeInstructions tests the output formatting of device code instructions
func TestPrintDeviceCodeInstructions(t *testing.T) {
	// Test case 1: With verificationURIComplete
	t.Run("With verification URI complete", func(t *testing.T) {
		// Redirect stdout for test
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		defer func() {
			os.Stdout = oldStdout
		}()

		deviceCode := &sso.DeviceCodeResponse{
			DeviceCode:              "device_code_123",
			UserCode:                "USER123",
			VerificationURI:         "https://example.com/verify",
			VerificationURIComplete: "https://example.com/verify?code=USER123",
			ExpiresIn:               300,
			Interval:                5,
		}
		providerName := "TestProvider"

		printDeviceCodeInstructions(deviceCode, providerName, w)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that output contains all expected elements
		assert.Contains(t, output, "https://example.com/verify", "Output should contain verification URI")
		assert.Contains(t, output, "USER123", "Output should contain user code")
		assert.Contains(t, output, "https://example.com/verify?code=USER123", "Output should contain complete verification URI")
		assert.Contains(t, output, "TestProvider", "Output should contain provider name")
	})

	// Test case 2: Without verificationURIComplete
	t.Run("Without verification URI complete", func(t *testing.T) {
		// Redirect stdout for test
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		defer func() {
			os.Stdout = oldStdout
		}()

		deviceCode := &sso.DeviceCodeResponse{
			DeviceCode:      "device_code_123",
			UserCode:        "USER123",
			VerificationURI: "https://example.com/verify",
			ExpiresIn:       300,
			Interval:        5,
		}
		providerName := "AnotherProvider"

		printDeviceCodeInstructions(deviceCode, providerName, w)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Check that output contains all expected elements but not the complete URI
		assert.Contains(t, output, "https://example.com/verify", "Output should contain verification URI")
		assert.Contains(t, output, "USER123", "Output should contain user code")
		assert.Contains(t, output, "AnotherProvider", "Output should contain provider name")
		assert.NotContains(t, output, "Or, open this URL in your browser", "Output should not mention alternative URL")
	})
}

// TestEndpointToProfileName tests conversion of endpoint URLs to profile names
func TestEndpointToProfileName(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "simple https URL",
			endpoint: "https://example.com",
			expected: "example_com",
		},
		{
			name:     "URL with port",
			endpoint: "https://example.com:8080",
			expected: "example_com_8080",
		},
		{
			name:     "URL with subdomain",
			endpoint: "https://api.prod.example.com",
			expected: "api_prod_example_com",
		},
		{
			name:     "URL with subdomain and port",
			endpoint: "https://api.prod.example.com:443",
			expected: "api_prod_example_com_443",
		},
		{
			name:     "http URL",
			endpoint: "http://localhost:1234",
			expected: "localhost_1234",
		},
		{
			name:     "URL with path (path should be ignored)",
			endpoint: "https://example.com:8080/api/v1",
			expected: "example_com_8080",
		},
		{
			name:     "URL with hyphen in hostname",
			endpoint: "https://my-api.example.com",
			expected: "my_api_example_com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := endpointToProfileName(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizeProfileName tests profile name sanitization
func TestSanitizeProfileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "myprofile",
			expected: "myprofile",
		},
		{
			name:     "name with dots",
			input:    "api.example.com",
			expected: "api_example_com",
		},
		{
			name:     "name with hyphens",
			input:    "my-api-server",
			expected: "my_api_server",
		},
		{
			name:     "name with consecutive underscores",
			input:    "my___profile",
			expected: "my_profile",
		},
		{
			name:     "name with leading/trailing underscores",
			input:    "_myprofile_",
			expected: "myprofile",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "default",
		},
		{
			name:     "URL-like string",
			input:    "https://example.com:8080",
			expected: "https_example_com_8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeProfileName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSaveTokenToProfile tests that SSO tokens can be saved to profiles
func TestSaveTokenToProfile(t *testing.T) {
	// Create a temporary directory for profiles
	tmpDir, err := os.MkdirTemp("", "bacalhau-sso-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create store and loader
	store := profile.NewStore(profilesDir)
	loader := profile.NewLoader(store, "", "")

	// Test data
	apiURL := "https://api.example.com:8080"
	token := "test-access-token-12345"
	profileName := endpointToProfileName(apiURL)

	// LoadOrCreate should create a new profile
	p, err := loader.LoadOrCreate(profileName, apiURL)
	require.NoError(t, err)
	assert.Equal(t, apiURL, p.Endpoint)
	assert.Nil(t, p.Auth, "Auth should be nil for new profile")

	// Set the auth token
	p.Auth = &profile.AuthConfig{Token: token}
	err = store.Save(profileName, p)
	require.NoError(t, err)

	// Verify profile was saved correctly
	loadedProfile, err := store.Load(profileName)
	require.NoError(t, err)
	assert.Equal(t, apiURL, loadedProfile.Endpoint)
	require.NotNil(t, loadedProfile.Auth)
	assert.Equal(t, token, loadedProfile.Auth.Token)

	// Verify the profile file exists
	assert.True(t, store.Exists(profileName))
}

// TestSaveTokenToExistingProfile tests updating an existing profile with SSO token
func TestSaveTokenToExistingProfile(t *testing.T) {
	// Create a temporary directory for profiles
	tmpDir, err := os.MkdirTemp("", "bacalhau-sso-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	profilesDir := filepath.Join(tmpDir, "profiles")

	// Create store and loader
	store := profile.NewStore(profilesDir)
	loader := profile.NewLoader(store, "", "")

	// Test data
	apiURL := "https://api.example.com:8080"
	profileName := endpointToProfileName(apiURL)

	// Create an existing profile with some settings
	existingProfile := &profile.Profile{
		Endpoint:    apiURL,
		Description: "My existing profile",
		Timeout:     "60s",
		TLS:         &profile.TLSConfig{Insecure: true},
	}
	err = store.Save(profileName, existingProfile)
	require.NoError(t, err)

	// LoadOrCreate should load the existing profile
	p, err := loader.LoadOrCreate(profileName, apiURL)
	require.NoError(t, err)
	assert.Equal(t, "My existing profile", p.Description)
	assert.Equal(t, "60s", p.Timeout)
	assert.True(t, p.IsInsecure())

	// Update with SSO token
	newToken := "new-sso-token-67890"
	p.Auth = &profile.AuthConfig{Token: newToken}
	err = store.Save(profileName, p)
	require.NoError(t, err)

	// Verify the profile retained original settings and has new token
	loadedProfile, err := store.Load(profileName)
	require.NoError(t, err)
	assert.Equal(t, apiURL, loadedProfile.Endpoint)
	assert.Equal(t, "My existing profile", loadedProfile.Description)
	assert.Equal(t, "60s", loadedProfile.Timeout)
	assert.True(t, loadedProfile.IsInsecure())
	require.NotNil(t, loadedProfile.Auth)
	assert.Equal(t, newToken, loadedProfile.Auth.Token)
}

// TestSetCurrentProfileOnFirstLogin tests that current profile is set on first SSO login
func TestSetCurrentProfileOnFirstLogin(t *testing.T) {
	// Create a temporary directory for profiles
	tmpDir, err := os.MkdirTemp("", "bacalhau-sso-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	profilesDir := filepath.Join(tmpDir, "profiles")
	store := profile.NewStore(profilesDir)
	loader := profile.NewLoader(store, "", "")

	// Test data
	apiURL := "https://api.example.com:8080"
	token := "test-token"
	profileName := endpointToProfileName(apiURL)

	// Verify no current profile exists
	current, err := store.GetCurrent()
	require.NoError(t, err)
	assert.Empty(t, current)

	// Create profile and set token
	p, err := loader.LoadOrCreate(profileName, apiURL)
	require.NoError(t, err)
	p.Auth = &profile.AuthConfig{Token: token}
	err = store.Save(profileName, p)
	require.NoError(t, err)

	// Set as current since no current exists
	if current, _ := store.GetCurrent(); current == "" {
		err = store.SetCurrent(profileName)
		require.NoError(t, err)
	}

	// Verify current is now set
	current, err = store.GetCurrent()
	require.NoError(t, err)
	assert.Equal(t, profileName, current)
}

// TestDoNotOverrideExistingCurrentProfile tests that existing current profile is not overridden
func TestDoNotOverrideExistingCurrentProfile(t *testing.T) {
	// Create a temporary directory for profiles
	tmpDir, err := os.MkdirTemp("", "bacalhau-sso-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	profilesDir := filepath.Join(tmpDir, "profiles")
	store := profile.NewStore(profilesDir)
	loader := profile.NewLoader(store, "", "")

	// Create and set an existing current profile
	existingProfileName := "existing_profile"
	existingProfile := &profile.Profile{
		Endpoint: "https://existing.example.com",
	}
	err = store.Save(existingProfileName, existingProfile)
	require.NoError(t, err)
	err = store.SetCurrent(existingProfileName)
	require.NoError(t, err)

	// Verify current profile is set
	current, err := store.GetCurrent()
	require.NoError(t, err)
	assert.Equal(t, existingProfileName, current)

	// Now "login" to a different endpoint
	apiURL := "https://api.example.com:8080"
	token := "test-token"
	profileName := endpointToProfileName(apiURL)

	p, err := loader.LoadOrCreate(profileName, apiURL)
	require.NoError(t, err)
	p.Auth = &profile.AuthConfig{Token: token}
	err = store.Save(profileName, p)
	require.NoError(t, err)

	// Only set current if none exists (mimic login.go behavior)
	if current, _ := store.GetCurrent(); current == "" {
		_ = store.SetCurrent(profileName)
	}

	// Verify original current profile is still set
	current, err = store.GetCurrent()
	require.NoError(t, err)
	assert.Equal(t, existingProfileName, current, "Current profile should not be overridden")
}
