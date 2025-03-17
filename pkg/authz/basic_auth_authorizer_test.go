package authz

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/credsecurity"
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

	t.Run("ValidBcryptPassword", func(t *testing.T) {
		// Setup an authorizer with a user that has a bcrypt hashed password
		authorizer := createTestAuthorizer()

		// Create a bcrypt hash of "securepass"
		bcryptManager := credsecurity.NewDefaultBcryptManager()
		hashedPassword, err := bcryptManager.HashPassword("securepass")
		require.NoError(t, err)

		// Add a user with a bcrypt hashed password
		authorizer.basicAuthUsers["bcryptuser"] = types.AuthUser{
			Alias:    "Bcrypt User",
			Password: hashedPassword,
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job"},
				},
			},
		}

		// Create valid auth header with the plaintext password
		authHeader := createBasicAuthHeader("bcryptuser", "securepass")

		// Execute
		user, authenticated, err := authorizer.validateBasicAuth(authHeader)

		// Verify
		require.NoError(t, err)
		assert.True(t, authenticated)
		assert.Equal(t, "Bcrypt User", user.Alias)
		assert.Equal(t, "read:job", user.Capabilities[0].Actions[0])
	})

	t.Run("InvalidBcryptPassword", func(t *testing.T) {
		// Setup an authorizer with a user that has a bcrypt hashed password
		authorizer := createTestAuthorizer()

		// Create a bcrypt hash of "securepass"
		bcryptManager := credsecurity.NewDefaultBcryptManager()
		hashedPassword, err := bcryptManager.HashPassword("securepass")
		require.NoError(t, err)

		// Add a user with a bcrypt hashed password
		authorizer.basicAuthUsers["bcryptuser"] = types.AuthUser{
			Alias:    "Bcrypt User",
			Password: hashedPassword,
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job"},
				},
			},
		}

		// Create invalid auth header with the wrong password
		authHeader := createBasicAuthHeader("bcryptuser", "wrongpass")

		// Execute
		_, authenticated, err := authorizer.validateBasicAuth(authHeader)

		// Verify
		require.Error(t, err)
		assert.False(t, authenticated)
		assert.Contains(t, err.Error(), "invalid basic auth credentials")
	})

	t.Run("MixedAuthenticationMethods", func(t *testing.T) {
		// Setup an authorizer with both plaintext and bcrypt users
		authorizer := createTestAuthorizer()

		// Create a bcrypt hash of "securepass"
		bcryptManager := credsecurity.NewDefaultBcryptManager()
		hashedPassword, err := bcryptManager.HashPassword("securepass")
		require.NoError(t, err)

		// Add a user with a bcrypt hashed password
		authorizer.basicAuthUsers["bcryptuser"] = types.AuthUser{
			Alias:    "Bcrypt User",
			Password: hashedPassword,
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job"},
				},
			},
		}

		// Test plain text user still works
		plainAuthHeader := createBasicAuthHeader("testuser", "testpass")
		user, authenticated, err := authorizer.validateBasicAuth(plainAuthHeader)
		require.NoError(t, err)
		assert.True(t, authenticated)
		assert.Equal(t, "Test User", user.Alias)

		// Test bcrypt user works too
		bcryptAuthHeader := createBasicAuthHeader("bcryptuser", "securepass")
		bcryptUser, bcryptAuthenticated, bcryptErr := authorizer.validateBasicAuth(bcryptAuthHeader)
		require.NoError(t, bcryptErr)
		assert.True(t, bcryptAuthenticated)
		assert.Equal(t, "Bcrypt User", bcryptUser.Alias)
	})

	t.Run("BcryptFormatButInvalidHash", func(t *testing.T) {
		// Setup
		authorizer := createTestAuthorizer()

		// Add a user with an invalid bcrypt format (correct prefix but invalid hash)
		authorizer.basicAuthUsers["invalidhashuser"] = types.AuthUser{
			Alias:    "Invalid Hash User",
			Password: "$2a$10$invalidhashnotvalidbcrypt",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Create auth header
		authHeader := createBasicAuthHeader("invalidhashuser", "anypassword")

		// Execute
		_, authenticated, err := authorizer.validateBasicAuth(authHeader)

		// Verify
		require.Error(t, err)
		assert.False(t, authenticated)
		assert.Contains(t, err.Error(), "invalid basic auth credentials")
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
