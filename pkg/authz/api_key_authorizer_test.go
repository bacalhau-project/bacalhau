package authz

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiKeyValidation(t *testing.T) {
	t.Run("ValidApiKey", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create valid auth header with API key
		authHeader := createBearerAuthHeader("valid-api-key-123")

		// Execute
		user, authenticated, err := authorizer.validateAPIKey(authHeader)

		// Verify
		require.NoError(t, err)
		assert.True(t, authenticated)
		assert.Equal(t, "API User", user.Alias)
		assert.Equal(t, "read:job", user.Capabilities[0].Actions[0])
	})

	t.Run("InvalidApiKey", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create invalid auth header
		authHeader := createBearerAuthHeader("invalid-api-key")

		// Execute
		_, authenticated, err := authorizer.validateAPIKey(authHeader)

		// Verify
		require.Error(t, err)
		assert.False(t, authenticated)
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("EmptyApiKey", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create auth header with empty API key
		authHeader := createBearerAuthHeader("")

		// Execute
		_, authenticated, err := authorizer.validateAPIKey(authHeader)

		// Verify
		require.Error(t, err)
		assert.False(t, authenticated)
		assert.Contains(t, err.Error(), "empty API key provided")
	})
}

func TestApiKeyAuthorization(t *testing.T) {
	t.Run("AuthorizedWithRequiredCapabilities", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request with valid API key for jobs endpoint
		req := createTestApiKeyRequest("/v1/jobs", createBearerAuthHeader("valid-api-key-123"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
		assert.Empty(t, auth.Reason) // No reason needed for success
	})

	t.Run("UnauthorizedWithMissingCapabilities", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request to node endpoint with regular user auth (missing node access)
		req := createTestApiKeyRequest("/v1/admin", createBearerAuthHeader("valid-api-key-123"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.False(t, auth.Approved)
		assert.True(t, auth.TokenValid) // Token is valid, just lacks capabilities
		assert.Contains(t, auth.Reason, "user 'API User' does not have the required capability")
	})

	t.Run("AuthorizedWithAdminCapabilities", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request with admin API key for admin endpoint
		req := createTestApiKeyRequest("/v1/admin", createBearerAuthHeader("admin-api-key-456"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
		assert.Empty(t, auth.Reason) // No reason needed for success
	})

	t.Run("SuccessForOpenEndpoint", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizerWithOpenEndpoint()

		// Create request to an open endpoint without auth
		req := createTestApiKeyRequest("/v1/open", "")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
		assert.Empty(t, auth.Reason) // No reason needed for open endpoints
	})

	t.Run("MissingAuthorizationHeader", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request without auth header
		req := createTestApiKeyRequest("/v1/jobs", "")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Equal(t, "Missing Authorization header", auth.Reason)
	})

	t.Run("MissingURL", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request without URL
		req := &http.Request{
			Header: http.Header{},
		}
		req.Header.Add("Authorization", createBearerAuthHeader("valid-api-key-123"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Equal(t, "Missing Request URL", auth.Reason)
	})

	// Add new test cases for API key specific scenarios
	t.Run("InvalidAPIKey", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request with invalid API key
		req := createTestApiKeyRequest("/v1/jobs", createBearerAuthHeader("invalid-api-key"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Equal(t, "invalid API key", auth.Reason)
	})

	t.Run("EmptyAPIKey", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request with empty API key
		req := createTestApiKeyRequest("/v1/jobs", createBearerAuthHeader(""))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Equal(t, "empty API key provided", auth.Reason)
	})

	t.Run("NonBearerAuthScheme", func(t *testing.T) {
		// Setup
		authorizer := createTestApiKeyAuthorizer()

		// Create request with wrong auth scheme
		req := createTestApiKeyRequest("/v1/jobs", "Basic sometoken")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Equal(t, "invalid API key", auth.Reason)
	})
}

func TestGetUserIdentifierForAPIKey(t *testing.T) {
	// Setup
	authorizer := createTestApiKeyAuthorizer()

	t.Run("User with Alias", func(t *testing.T) {
		user := types.AuthUser{
			Alias:    "Test Alias",
			Username: "testuser",
			APIKey:   "test-api-key-12345",
		}
		identifier := authorizer.getUserIdentifier(user)
		assert.Equal(t, "Test Alias", identifier)
	})

	t.Run("User with Username but no Alias", func(t *testing.T) {
		user := types.AuthUser{
			Username: "testuser",
			APIKey:   "test-api-key-12345",
		}
		identifier := authorizer.getUserIdentifier(user)
		assert.Equal(t, "testuser", identifier)
	})

	t.Run("User with only API Key", func(t *testing.T) {
		user := types.AuthUser{
			APIKey: "test-api-key-12345",
		}
		identifier := authorizer.getUserIdentifier(user)
		assert.Equal(t, "API key ending in ...12345", identifier)
	})

	t.Run("User with short API Key", func(t *testing.T) {
		user := types.AuthUser{
			APIKey: "1234",
		}
		identifier := authorizer.getUserIdentifier(user)
		assert.Equal(t, "API key 1234", identifier)
	})

	t.Run("Unknown User", func(t *testing.T) {
		user := types.AuthUser{}
		identifier := authorizer.getUserIdentifier(user)
		assert.Equal(t, "unknown user", identifier)
	})
}

// Helper functions to create test data

func createTestApiKeyAuthorizer() *apiKeyAuthorizer {
	// Create API key users map
	apiKeyUsers := map[string]types.AuthUser{
		"valid-api-key-123": {
			Alias:  "API User",
			APIKey: "valid-api-key-123",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job"},
				},
			},
		},
		"admin-api-key-456": {
			Alias:  "Admin API User",
			APIKey: "admin-api-key-456",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job", "read:node", "write:node"},
				},
			},
		},
	}

	// Create capability checker
	capabilityChecker := NewCapabilityChecker()

	// Create endpoint permissions
	endpointPermissions := map[string]string{
		"/v1/jobs":  "job",
		"/v1/admin": "node",
	}

	// Return the authorizer
	return &apiKeyAuthorizer{
		nodeID:              "test-node",
		apiKeyUsers:         apiKeyUsers,
		capabilityChecker:   capabilityChecker,
		endpointPermissions: endpointPermissions,
	}
}

func createTestApiKeyAuthorizerWithOpenEndpoint() *apiKeyAuthorizer {
	authorizer := createTestApiKeyAuthorizer()
	authorizer.endpointPermissions["/v1/open"] = "open"
	return authorizer
}

func createBearerAuthHeader(apiKey string) string {
	return "Bearer " + apiKey
}

func createTestApiKeyRequest(path, authHeader string) *http.Request {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:1234%s", path), nil)
	req.URL, _ = url.Parse(fmt.Sprintf("http://localhost:1234%s", path))

	if authHeader != "" {
		req.Header = http.Header{}
		req.Header.Add("Authorization", authHeader)
	}

	return req
}
