package sso

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// setupMockServer creates a test server that mocks OAuth2 endpoints
func setupMockServer(t *testing.T, deviceAuthHandler, tokenHandler http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()

	// Only register handlers that are not nil
	if deviceAuthHandler != nil {
		mux.HandleFunc("/device/code", deviceAuthHandler)
	}

	if tokenHandler != nil {
		mux.HandleFunc("/token", tokenHandler)
	}

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// createTestConfig creates a test OAuth2 config with the provided server URL
func createTestConfig(serverURL string) types.Oauth2Config {
	return types.Oauth2Config{
		DeviceClientId:              "test-client-id",
		Scopes:                      []string{"test-scope"},
		DeviceAuthorizationEndpoint: serverURL + "/device/code",
		TokenEndpoint:               serverURL + "/token",
		Audience:                    "test-audience",
	}
}

// TestNewOAuth2Service tests the creation of a new OAuth2Service
func TestNewOAuth2Service(t *testing.T) {
	// Create a test configuration
	config := types.Oauth2Config{
		DeviceClientId:              "test-client-id",
		Scopes:                      []string{"profile", "email"},
		DeviceAuthorizationEndpoint: "https://example.com/device/code",
		TokenEndpoint:               "https://example.com/token",
		Audience:                    "test-audience",
	}

	// Create a new OAuth2Service
	service := NewOAuth2Service(config)

	// Verify the service was created with the correct configuration
	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.Equal(t, "test-client-id", service.oauthConfig.ClientID)
	assert.Equal(t, []string{"profile", "email"}, service.oauthConfig.Scopes)
	assert.Equal(t, "https://example.com/token", service.oauthConfig.Endpoint.TokenURL)
	assert.Equal(t, "https://example.com/device/code", service.oauthConfig.Endpoint.DeviceAuthURL)
	assert.Equal(t, oauth2.AuthStyleInParams, service.oauthConfig.Endpoint.AuthStyle)
}

// TestInitiateDeviceCodeFlow tests the device code flow initiation
func TestInitiateDeviceCodeFlow(t *testing.T) {
	// Set up a test server to mock the device authorization endpoint
	server := setupMockServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			// Verify the request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			err := r.ParseForm()
			require.NoError(t, err)
			assert.Equal(t, "test-client-id", r.Form.Get("client_id"))
			assert.Equal(t, "test-scope", r.Form.Get("scope"))
			assert.Equal(t, "test-audience", r.Form.Get("audience"))

			// Send a mock response
			w.Header().Set("Content-Type", "application/json")
			expiry := time.Now().Add(5 * time.Minute)
			response := oauth2.DeviceAuthResponse{
				DeviceCode:              "device-code-123",
				UserCode:                "USER-123",
				VerificationURI:         "https://example.com/verify",
				VerificationURIComplete: "https://example.com/verify?code=USER-123",
				Expiry:                  expiry,
				Interval:                5,
			}

			json.NewEncoder(w).Encode(response)
		},
		nil, // Token handler not used in this test
	)

	// Create a test configuration with the test server URL
	config := createTestConfig(server.URL)

	// Create a new OAuth2Service with the test configuration
	service := NewOAuth2Service(config)

	// Call InitiateDeviceCodeFlow
	ctx := context.Background()
	response, err := service.InitiateDeviceCodeFlow(ctx)

	// Verify the response
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "device-code-123", response.DeviceCode)
	assert.Equal(t, "USER-123", response.UserCode)
	assert.Equal(t, "https://example.com/verify", response.VerificationURI)
	assert.Equal(t, "https://example.com/verify?code=USER-123", response.VerificationURIComplete)
	assert.GreaterOrEqual(t, response.ExpiresIn, 290) // Approximately 5 minutes in seconds
	assert.Equal(t, 5, response.Interval)
}

// TestInitiateDeviceCodeFlowError tests error handling in device code flow initiation
func TestInitiateDeviceCodeFlowError(t *testing.T) {
	// Set up a test server that returns an error response
	server := setupMockServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			// Return an error response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "invalid_request",
				"error_description": "Invalid client ID",
			})
		},
		nil, // Token handler not used in this test
	)

	// Create a test configuration with the test server URL
	config := createTestConfig(server.URL)

	// Create a new OAuth2Service with the test configuration
	service := NewOAuth2Service(config)

	// Call InitiateDeviceCodeFlow
	ctx := context.Background()
	response, err := service.InitiateDeviceCodeFlow(ctx)

	// Verify the error
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "device authorization request failed")
}

// TestPollForToken tests successful token polling
func TestPollForToken(t *testing.T) {
	// Set up a test server to mock the token endpoint
	server := setupMockServer(t,
		nil, // Device auth handler not used in this test
		func(w http.ResponseWriter, r *http.Request) {
			// Verify the request
			assert.Equal(t, "POST", r.Method)

			err := r.ParseForm()
			require.NoError(t, err)
			assert.Equal(t, "test-client-id", r.Form.Get("client_id"))
			assert.Equal(t, "urn:ietf:params:oauth:grant-type:device_code", r.Form.Get("grant_type"))
			assert.Equal(t, "device-code-123", r.Form.Get("device_code"))

			// Send a mock token response
			w.Header().Set("Content-Type", "application/json")

			// Create response JSON directly
			response := map[string]interface{}{
				"access_token":  "access-token-123",
				"token_type":    "Bearer",
				"refresh_token": "refresh-token-123",
				"expires_in":    3600,
				"id_token":      "id-token-123",
			}

			err = json.NewEncoder(w).Encode(response)
			require.NoError(t, err)
		},
	)

	// Create a test configuration with the test server URL
	config := createTestConfig(server.URL)

	// Create a new OAuth2Service with the test configuration
	service := NewOAuth2Service(config)

	// Call PollForToken
	ctx := context.Background()
	token, err := service.PollForToken(ctx, "device-code-123")

	// Verify the response
	require.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "access-token-123", token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.Equal(t, "refresh-token-123", token.RefreshToken)
	assert.True(t, token.Expiry.After(time.Now()))
	assert.Equal(t, "id-token-123", token.Extra("id_token"))
}

// TestPollForTokenError tests error handling in token polling
func TestPollForTokenError(t *testing.T) {
	// Set up a test server that returns an error response
	server := setupMockServer(t,
		nil, // Device auth handler not used in this test
		func(w http.ResponseWriter, r *http.Request) {
			// Return an error response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "authorization_pending",
				"error_description": "The authorization request is still pending",
			})
		},
	)

	// Create a test configuration with the test server URL
	config := createTestConfig(server.URL)

	// Create a new OAuth2Service with the test configuration
	service := NewOAuth2Service(config)

	// Create a context with a timeout to prevent the test from hanging
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Call PollForToken
	token, err := service.PollForToken(ctx, "device-code-123")

	// Verify the error
	assert.Error(t, err)
	assert.Nil(t, token)
	// Context deadline error is expected due to the timeout
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

// TestPollForTokenCancellation tests cancellation during token polling
func TestPollForTokenCancellation(t *testing.T) {
	// Set up a test server that delays the response
	server := setupMockServer(t,
		nil, // Device auth handler not used in this test
		func(w http.ResponseWriter, r *http.Request) {
			// Sleep to simulate waiting
			time.Sleep(500 * time.Millisecond)

			// Return a response (should not be reached due to cancelled context)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "authorization_pending",
				"error_description": "The authorization request is still pending",
			})
		},
	)

	// Create a test configuration with the test server URL
	config := createTestConfig(server.URL)

	// Create a new OAuth2Service with the test configuration
	service := NewOAuth2Service(config)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start a goroutine to cancel the context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Call PollForToken
	token, err := service.PollForToken(ctx, "device-code-123")

	// Verify the error
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Contains(t, err.Error(), "context canceled")
}
