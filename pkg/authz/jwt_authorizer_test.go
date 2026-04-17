package authz

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupJWKSServer creates a test JWKS server with the provided key
func setupJWKSServer(t *testing.T, rsaKey *rsa.PrivateKey) *httptest.Server {
	// Create a JWKS response with the public key
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": "test-key-id",
				"use": "sig",
				"alg": "RS256",
				"n":   base64URLEncode(rsaKey.Public().(*rsa.PublicKey).N.Bytes()),
				"e":   base64URLEncode([]byte{1, 0, 1}), // Standard RSA exponent 65537 in bytes
			},
		},
	}

	// Create a test server that returns the JWKS
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// base64URLEncode encodes bytes to base64URL format
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// generateTestToken generates a JWT token for testing
func generateTestToken(t *testing.T, rsaKey *rsa.PrivateKey, claims *JWTClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-id"

	tokenString, err := token.SignedString(rsaKey)
	require.NoError(t, err, "Failed to sign token")

	return tokenString
}

// TestNewJWTAuthorizer_Success tests successful creation of a JWT authorizer
func TestNewJWTAuthorizer_Success(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create capability checker
	capChecker := NewCapabilityChecker()

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize authorizer
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)

	// Check results
	assert.NoError(t, err)
	assert.NotNil(t, auth)
}

// TestNewJWTAuthorizer_MissingJWKSURL tests authorizer creation with missing JWKS URL
func TestNewJWTAuthorizer_MissingJWKSURL(t *testing.T) {
	// Create auth config with empty JWKS URL
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			// JWKSUri is intentionally empty
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create capability checker
	capChecker := NewCapabilityChecker()

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize authorizer - should fail
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)

	// Check results
	assert.Error(t, err)
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "missing required OAuth2 fields for JWT authorization: JWKSUri")
}

// TestAuthorize_ValidToken tests authorization with a valid token
func TestAuthorize_ValidToken(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize capability checker and authorizer
	capChecker := NewCapabilityChecker()
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Create JWT claims
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   "test-user",
			Audience:  jwt.ClaimStrings{"test-audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Permissions: []string{"write:job"},
	}

	// Generate token
	tokenString := generateTestToken(t, rsaKey, claims)

	// Create test request
	req := httptest.NewRequest("POST", "http://example.com/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Call Authorize
	result, err := auth.Authorize(req)

	// Check results
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.True(t, result.TokenValid)
	assert.Empty(t, result.Reason) // No reason needed for success
}

// TestAuthorize_ExpiredToken tests authorization with an expired token
func TestAuthorize_ExpiredToken(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize capability checker and authorizer
	capChecker := NewCapabilityChecker()
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Create JWT claims with expired token
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   "test-user",
			Audience:  jwt.ClaimStrings{"test-audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
		Permissions: []string{"write:job"},
	}

	// Generate token
	tokenString := generateTestToken(t, rsaKey, claims)

	// Create test request
	req := httptest.NewRequest("POST", "http://example.com/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Call Authorize
	result, err := auth.Authorize(req)

	// Check results
	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.False(t, result.TokenValid)
	assert.Equal(t, "invalid JWT token", result.Reason)
}

// TestAuthorize_WrongIssuer tests authorization with token from wrong issuer
func TestAuthorize_WrongIssuer(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize capability checker and authorizer
	capChecker := NewCapabilityChecker()
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Create JWT claims with wrong issuer
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "wrong-issuer", // Wrong issuer
			Subject:   "test-user",
			Audience:  jwt.ClaimStrings{"test-audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Permissions: []string{"write:job"},
	}

	// Generate token
	tokenString := generateTestToken(t, rsaKey, claims)

	// Create test request
	req := httptest.NewRequest("POST", "http://example.com/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Call Authorize
	result, err := auth.Authorize(req)

	// Check results
	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.False(t, result.TokenValid)
	assert.Equal(t, "invalid JWT token", result.Reason)
}

// TestAuthorize_MissingPermission tests authorization with insufficient permissions
func TestAuthorize_MissingPermission(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize capability checker and authorizer
	capChecker := NewCapabilityChecker()
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Create JWT claims with only read permission when write is needed
	claims := &JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   "test-user",
			Audience:  jwt.ClaimStrings{"test-audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Permissions: []string{"read:job"}, // Only has read permission
	}

	// Generate token
	tokenString := generateTestToken(t, rsaKey, claims)

	// Create POST request which requires write permission
	req := httptest.NewRequest("POST", "http://example.com/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Call Authorize
	result, err := auth.Authorize(req)

	// Check results
	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.True(t, result.TokenValid) // Token is valid but lacks required permissions
	assert.Contains(t, result.Reason, "user 'test-user' does not have the required capability")
}

// TestAuthorize_OpenEndpoint tests authorization for an open endpoint
func TestAuthorize_OpenEndpoint(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create endpoint permissions with open endpoint
	endpointPerms := map[string]string{
		"/api/open": string(ResourceTypeOpen),
	}

	// Initialize capability checker and authorizer
	capChecker := NewCapabilityChecker()
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Create test request to open endpoint (no token needed)
	req := httptest.NewRequest("GET", "http://example.com/api/open", nil)
	// No Authorization header set

	// Call Authorize
	result, err := auth.Authorize(req)

	// Check results
	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.True(t, result.TokenValid)
	assert.Empty(t, result.Reason) // No reason needed for open endpoints
}

// TestAuthorize_MissingAuthHeader tests authorization with missing Authorization header
func TestAuthorize_MissingAuthHeader(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize capability checker and authorizer
	capChecker := NewCapabilityChecker()
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Create test request without Authorization header
	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	// No Authorization header set

	// Call Authorize
	result, err := auth.Authorize(req)

	// Check results
	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.False(t, result.TokenValid)
	assert.Equal(t, "Missing Authorization header", result.Reason)
}

// TestAuthorize_InvalidAuthHeaderFormat tests authorization with invalid Authorization header format
func TestAuthorize_InvalidAuthHeaderFormat(t *testing.T) {
	// Generate an RSA key for signing
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Setup JWKS server
	jwksServer := setupJWKSServer(t, rsaKey)

	// Create auth config
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        jwksServer.URL,
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize capability checker and authorizer
	capChecker := NewCapabilityChecker()
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Create test request with invalid Authorization header format
	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token123")

	// Call Authorize
	result, err := auth.Authorize(req)

	// Check results
	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.False(t, result.TokenValid)
	assert.Equal(t, "invalid authorization header format, expected 'Bearer TOKEN'", result.Reason)
}

// TestNewJWTAuthorizer_EmptyOAuth2Config tests the case when OAuth2 config is completely empty
func TestNewJWTAuthorizer_EmptyOAuth2Config(t *testing.T) {
	// Create auth config with empty OAuth2 config
	authConfig := types.AuthConfig{
		// Empty OAuth2 config
	}

	// Create capability checker
	capChecker := NewCapabilityChecker()

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize authorizer - should return DenyAuthorizer, not error
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)

	// Check results
	assert.NoError(t, err, "Empty OAuth2 config should not return an error")
	assert.NotNil(t, auth, "Should return a DenyAuthorizer")

	// Verify it's a DenyAuthorizer by calling Authorize and checking the response
	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)

	result, err := auth.Authorize(req)

	assert.NoError(t, err)
	assert.False(t, result.Approved)
	assert.False(t, result.TokenValid)
	assert.Contains(t, result.Reason, "OAuth2 authentication not configured on server")
}

// TestNewJWTAuthorizer_PartialOAuth2Config tests the case when OAuth2 config is partially filled
func TestNewJWTAuthorizer_PartialOAuth2Config(t *testing.T) {
	// Create auth config with partial OAuth2 config (missing some required fields)
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri: "https://example.com/.well-known/jwks.json",
			// Missing Issuer, Audience, DeviceClientID
		},
	}

	// Create capability checker
	capChecker := NewCapabilityChecker()

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize authorizer - should fail with specific error
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)

	// Check results
	assert.Error(t, err, "Partial OAuth2 config should return an error")
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "missing required OAuth2 fields")
	assert.Contains(t, err.Error(), "Issuer")
	assert.Contains(t, err.Error(), "Audience")
	assert.Contains(t, err.Error(), "DeviceClientID")
}

// TestNewJWTAuthorizer_InvalidJWKSURL tests the case when JWKS URL is invalid
func TestNewJWTAuthorizer_InvalidJWKSURL(t *testing.T) {
	// Create auth config with invalid JWKS URL
	authConfig := types.AuthConfig{
		Oauth2: types.Oauth2Config{
			JWKSUri:        "invalid-url", // Not a valid URL
			Issuer:         "test-issuer",
			Audience:       "test-audience",
			DeviceClientID: "test-client-id",
		},
	}

	// Create capability checker
	capChecker := NewCapabilityChecker()

	// Create endpoint permissions
	endpointPerms := map[string]string{
		"/api/test": string(ResourceTypeJob),
	}

	// Initialize authorizer - should fail with specific error
	auth, err := NewJWTAuthorizer(
		context.Background(),
		"test-node-id",
		authConfig,
		capChecker,
		endpointPerms,
	)

	// Check results
	assert.Error(t, err, "Invalid JWKS URL should return an error")
	assert.Nil(t, auth)
	assert.Contains(t, err.Error(), "invalid JWKS URL format")
}

// TestDenyAuthorizer tests the behavior of the DenyAuthorizer
func TestDenyAuthorizer(t *testing.T) {
	t.Run("DefaultReason", func(t *testing.T) {
		// Create a DenyAuthorizer with default reason
		auth := NewDenyAuthorizer("")

		// Create test request
		req := httptest.NewRequest("GET", "http://example.com/api/test", nil)

		// Call Authorize
		result, err := auth.Authorize(req)

		// Check results
		require.NoError(t, err)
		assert.False(t, result.Approved)
		assert.False(t, result.TokenValid)
		assert.Equal(t, "access denied by policy", result.Reason)
	})

	t.Run("CustomReason", func(t *testing.T) {
		// Create a DenyAuthorizer with custom reason
		customReason := "custom denial reason"
		auth := NewDenyAuthorizer(customReason)

		// Create test request
		req := httptest.NewRequest("GET", "http://example.com/api/test", nil)

		// Call Authorize
		result, err := auth.Authorize(req)

		// Check results
		require.NoError(t, err)
		assert.False(t, result.Approved)
		assert.False(t, result.TokenValid)
		assert.Equal(t, customReason, result.Reason)
	})
}
