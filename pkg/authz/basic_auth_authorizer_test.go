package authz

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicAuthValidation(t *testing.T) {
	t.Run("ValidCredentials", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Create valid auth header
		authHeader := createBasicAuthHeader("testuser", "testpass")

		// Execute
		user, authenticated, err := authorizer.validateBasicAuth(authHeader)

		// Verify
		require.NoError(t, err)
		assert.True(t, authenticated)
		assert.Equal(t, "Test User", user.Alias)
		assert.Equal(t, "read:job", user.Capabilities[0].Actions[0])
	})

	t.Run("InvalidUsername", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Create invalid auth header
		authHeader := createBasicAuthHeader("wronguser", "testpass")

		// Execute
		_, authenticated, err := authorizer.validateBasicAuth(authHeader)

		// Verify
		require.Error(t, err)
		assert.False(t, authenticated)
		assert.Contains(t, err.Error(), "invalid basic auth credentials")
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Create invalid auth header
		authHeader := createBasicAuthHeader("testuser", "wrongpass")

		// Execute
		_, authenticated, err := authorizer.validateBasicAuth(authHeader)

		// Verify
		require.Error(t, err)
		assert.False(t, authenticated)
		assert.Contains(t, err.Error(), "invalid basic auth credentials")
	})

	t.Run("MalformedHeader", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Malformed header (missing colon)
		credentials := base64.StdEncoding.EncodeToString([]byte("testuser"))
		authHeader := "Basic " + credentials

		// Execute
		_, authenticated, err := authorizer.validateBasicAuth(authHeader)

		// Verify
		require.Error(t, err)
		assert.False(t, authenticated)
		assert.Contains(t, err.Error(), "invalid basic auth credentials format")
	})
}

func TestBasicAuthAuthorization(t *testing.T) {
	t.Run("AuthorizedWithRequiredCapabilities", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Create request with valid auth for jobs endpoint
		req := createTestRequest("/v1/jobs", createBasicAuthHeader("testuser", "testpass"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
	})

	t.Run("UnauthorizedWithMissingCapabilities", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Create request to node endpoint with regular user auth (missing node access)
		req := createTestRequest("/v1/admin", createBasicAuthHeader("testuser", "testpass"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.Error(t, err)
		assert.False(t, auth.Approved)
		assert.True(t, auth.TokenValid) // Token is valid, just lacks capabilities
		assert.Contains(t, err.Error(), "does not have the required capability")
	})

	t.Run("SuccessForOpenEndpoint", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizerWithOpenEndpoint()

		// Create request to an open endpoint without auth
		req := createTestRequest("/v1/open", "")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
	})

	t.Run("MissingAuthorizationHeader", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Create request without auth header
		req := createTestRequest("/v1/jobs", "")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.Error(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Contains(t, err.Error(), "missing authorization header")
	})

	t.Run("MissingURL", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Create request without URL
		req := &http.Request{
			Header: http.Header{},
		}
		req.Header.Add("Authorization", createBasicAuthHeader("testuser", "testpass"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.Error(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Contains(t, err.Error(), "missing URL")
	})
}

// Helper functions to create test data

func createTestAuthorizer() *basicAuthAuthorizer {
	// Create basic auth users map
	basicAuthUsers := map[string]types.AuthUser{
		"testuser": {
			Alias:    "Test User",
			Password: "testpass",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job"},
				},
			},
		},
		"adminuser": {
			Alias:    "Admin User",
			Password: "adminpass",
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
	return &basicAuthAuthorizer{
		nodeID:              "test-node",
		basicAuthUsers:      basicAuthUsers,
		capabilityChecker:   capabilityChecker,
		endpointPermissions: endpointPermissions,
	}
}

func createTestAuthorizerWithOpenEndpoint() *basicAuthAuthorizer {
	authorizer := createTestAuthorizer()
	authorizer.endpointPermissions["/v1/open"] = "open"
	return authorizer
}

func createBasicAuthHeader(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func createTestRequest(path, authHeader string) *http.Request {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:1234%s", path), nil)
	req.URL, _ = url.Parse(fmt.Sprintf("http://localhost:1234%s", path))

	if authHeader != "" {
		req.Header = http.Header{}
		req.Header.Add("Authorization", authHeader)
	}

	return req
}
