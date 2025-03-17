package authz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryPointUserValidation(t *testing.T) {
	t.Run("ValidUserWithUsernamePassword", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:    "Test User",
			Username: "testuser",
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.NoError(t, err)
	})

	t.Run("ValidUserWithApiKey", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:  "API User",
			ApiKey: "this-is-a-valid-api-key-12345678901234567890",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.NoError(t, err)
	})

	t.Run("InvalidUserWithoutAlias", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Username: "testuser",
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty Alias")
	})

	t.Run("InvalidUserWithBothAuthMethods", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:    "Dual Auth User",
			Username: "testuser",
			Password: "password1234567",
			ApiKey:   "this-is-a-valid-api-key-12345678901234567890",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has both username/password and API key")
	})

	t.Run("InvalidUserWithNoAuthMethod", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias: "No Auth User",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has neither username/password nor API key")
	})

	t.Run("InvalidUserWithNonAlphanumericUsername", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:    "Special Char User",
			Username: "user-with-hyphens",
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must contain only alphanumeric characters")
	})

	t.Run("InvalidUserWithShortPassword", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:    "Short Pass User",
			Username: "testuser",
			Password: "short", // Too short
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is too short")
	})

	t.Run("InvalidUserWithShortApiKey", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:  "Short API Key User",
			ApiKey: "tooshort", // Too short
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is too short")
	})

	t.Run("InvalidUserWithNoCapabilities", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:        "No Capabilities User",
			Username:     "testuser",
			Password:     "password1234567",
			Capabilities: []types.Capability{}, // Empty capabilities
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no capabilities defined")
	})

	t.Run("InvalidUserWithEmptyUsername", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:    "Empty Username User",
			Username: "", // Empty username
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has neither username/password nor API key")
	})

	t.Run("InvalidUserWithEmptyPassword", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		user := types.AuthUser{
			Alias:    "Empty Password User",
			Username: "testuser",
			Password: "", // Empty password
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has neither username/password nor API key")
	})

	t.Run("InvalidUserWithLongUsername", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		// Create a username longer than 100 characters
		longUsername := "a"
		for i := 0; i < 101; i++ {
			longUsername += "a"
		}

		user := types.AuthUser{
			Alias:    "Long Username User",
			Username: longUsername,
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("InvalidUserWithLongPassword", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		// Create a password longer than 100 characters
		longPassword := ""
		for i := 0; i < 300; i++ {
			longPassword += "a"
		}

		user := types.AuthUser{
			Alias:    "Long Password User",
			Username: "testuser",
			Password: longPassword,
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("InvalidUserWithLongApiKey", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		// Create an API key longer than 255 characters
		longApiKey := ""
		for i := 0; i < 256; i++ {
			longApiKey += "a"
		}

		user := types.AuthUser{
			Alias:  "Long API Key User",
			ApiKey: longApiKey,
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("ValidUserWithMaximumLengths", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}
		// Create values at the maximum allowed lengths
		username := ""
		for i := 0; i < 100; i++ {
			username += "a"
		}

		password := ""
		for i := 0; i < 100; i++ {
			password += "a"
		}

		user := types.AuthUser{
			Alias:    "Maximum Length User",
			Username: username,
			Password: password,
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.validateUser(user)

		// Verify
		require.NoError(t, err)
	})
}

func TestEntryPointDuplicateChecking(t *testing.T) {
	t.Run("DuplicateAlias", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}

		seenAliases := map[string]string{
			"test user": "Test User", // Use lowercase key as that's what the function checks
		}
		seenUsernames := map[string]string{}
		seenApiKeys := map[string]bool{}

		user := types.AuthUser{
			Alias:    "Test User", // Same as "Test User" when converted to lowercase
			Username: "anotheruser",
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.checkForDuplicates(user, seenAliases, seenUsernames, seenApiKeys)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate alias detected")
	})

	t.Run("DuplicateUsername", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}

		seenAliases := map[string]string{}
		seenUsernames := map[string]string{
			"testuser": "testuser",
		}
		seenApiKeys := map[string]bool{}

		user := types.AuthUser{
			Alias:    "Another User",
			Username: "TESTUSER", // Same as "testuser" when converted to lowercase
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.checkForDuplicates(user, seenAliases, seenUsernames, seenApiKeys)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate username detected")
	})

	t.Run("DuplicateApiKey", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}

		seenAliases := map[string]string{}
		seenUsernames := map[string]string{}
		seenApiKeys := map[string]bool{
			"this-is-a-valid-api-key-12345678901234567890": true,
		}

		user := types.AuthUser{
			Alias:  "Another API User",
			ApiKey: "this-is-a-valid-api-key-12345678901234567890", // Duplicate API key
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.checkForDuplicates(user, seenAliases, seenUsernames, seenApiKeys)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate API key detected")
	})

	t.Run("NoDuplicates", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}

		seenAliases := map[string]string{
			"existinguser": "Existing User",
		}
		seenUsernames := map[string]string{
			"existingusername": "existingUsername",
		}
		seenApiKeys := map[string]bool{
			"existing-api-key-12345678901234567890": true,
		}

		user := types.AuthUser{
			Alias:    "New User",
			Username: "newuser",
			Password: "password1234567",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job"},
				},
			},
		}

		// Execute
		err := authorizer.checkForDuplicates(user, seenAliases, seenUsernames, seenApiKeys)

		// Verify
		require.NoError(t, err)
	})
}

func TestEntryPointValidateAllUsers(t *testing.T) {
	t.Run("ValidUsers", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}

		users := []types.AuthUser{
			{
				Alias:    "Basic Auth User",
				Username: "basicuser",
				Password: "password1234567",
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job"},
					},
				},
			},
			{
				Alias:  "API Key User",
				ApiKey: "this-is-a-valid-api-key-12345678901234567890",
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job", "write:job"},
					},
				},
			},
		}

		// Execute
		err := authorizer.validateAllUsers(users)

		// Verify
		require.NoError(t, err)
	})

	t.Run("InvalidUser", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}

		users := []types.AuthUser{
			{
				Alias:    "Valid User",
				Username: "validuser",
				Password: "password1234567",
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job"},
					},
				},
			},
			{
				Alias: "Invalid User", // Missing auth method
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job"},
					},
				},
			},
		}

		// Execute
		err := authorizer.validateAllUsers(users)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "has neither username/password nor API key")
	})

	t.Run("DuplicateUsers", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{}

		users := []types.AuthUser{
			{
				Alias:    "Same Username",
				Username: "sameuser",
				Password: "password1234567",
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job"},
					},
				},
			},
			{
				Alias:    "Another User",
				Username: "SAMEUSER", // Same username but different case
				Password: "password1234567",
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job"},
					},
				},
			},
		}

		// Execute
		err := authorizer.validateAllUsers(users)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate username detected")
	})
}

func TestEntryPointPopulateUserMaps(t *testing.T) {
	t.Run("PopulateUserMaps", func(t *testing.T) {
		// Setup
		authorizer := &entryPointAuthorizer{
			basicAuthUsers: make(map[string]types.AuthUser),
			apiKeyUsers:    make(map[string]types.AuthUser),
		}

		users := []types.AuthUser{
			{
				Alias:    "Basic Auth User",
				Username: "basicuser",
				Password: "password1234567",
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job"},
					},
				},
			},
			{
				Alias:  "API Key User",
				ApiKey: "this-is-a-valid-api-key-12345678901234567890",
				Capabilities: []types.Capability{
					{
						Actions: []string{"read:job", "write:job"},
					},
				},
			},
		}

		// Execute
		authorizer.populateUserMaps(users)

		// Verify
		assert.Len(t, authorizer.basicAuthUsers, 1)
		assert.Len(t, authorizer.apiKeyUsers, 1)

		// Check basic auth user
		basicUser, exists := authorizer.basicAuthUsers["basicuser"]
		assert.True(t, exists)
		assert.Equal(t, "Basic Auth User", basicUser.Alias)

		// Check API key user
		apiUser, exists := authorizer.apiKeyUsers["this-is-a-valid-api-key-12345678901234567890"]
		assert.True(t, exists)
		assert.Equal(t, "API Key User", apiUser.Alias)
	})
}

func TestEntryPointAuthorize(t *testing.T) {
	t.Run("AuthorizeBasicAuth", func(t *testing.T) {
		// Setup
		authorizer := createTestEntryPointAuthorizer()

		// Create request with basic auth header
		req := createTestEntryPointRequest("/v1/jobs", createBasicAuthHeader("testuser", "testpass"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
	})

	t.Run("AuthorizeApiKey", func(t *testing.T) {
		// Setup
		authorizer := createTestEntryPointAuthorizer()

		// Create request with API key header
		req := createTestEntryPointRequest("/v1/jobs", createBearerAuthHeader("valid-api-key-123"))

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
	})

	t.Run("AuthorizeOpenEndpoint", func(t *testing.T) {
		// Setup
		authorizer := createTestEntryPointAuthorizer()

		// Create request to an open endpoint without auth
		req := createTestEntryPointRequest("/api/v1/version", "")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
	})

	t.Run("UnauthorizedWithUnsupportedAuthMethod", func(t *testing.T) {
		// Setup
		authorizer := createTestEntryPointAuthorizer()

		// Create request with unsupported auth header
		req := createTestEntryPointRequest("/v1/jobs", "Unsupported auth-method-here")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.Error(t, err)
		assert.False(t, auth.Approved)
		assert.False(t, auth.TokenValid)
		assert.Contains(t, err.Error(), "unsupported authentication method")
	})

	t.Run("MissingURL", func(t *testing.T) {
		// Setup
		authorizer := createTestEntryPointAuthorizer()

		// Create request without URL
		req := &http.Request{
			Header: http.Header{},
		}
		req.Header.Add("Authorization", createBasicAuthHeader("testuser", "testpass"))

		// Execute
		_, err := authorizer.Authorize(req)

		// Verify
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing URL")
	})

	t.Run("MissingAuthorizationHeader", func(t *testing.T) {
		// Setup
		authorizer := createTestEntryPointAuthorizer()

		// Create request without auth header
		req := createTestEntryPointRequest("/v1/jobs", "")

		// Execute
		auth, err := authorizer.Authorize(req)

		// Verify
		require.Error(t, err)
		assert.Equal(t, Authorization{}, auth)
		assert.Contains(t, err.Error(), "missing authorization header")
	})
}

func TestEntryPointCreation(t *testing.T) {
	t.Run("CreateEntryPointAuthorizer", func(t *testing.T) {
		// Setup
		authConfig := types.AuthConfig{
			Users: []types.AuthUser{
				{
					Alias:    "Basic Auth User",
					Username: "basicuser",
					Password: "password1234567",
					Capabilities: []types.Capability{
						{
							Actions: []string{"read:job"},
						},
					},
				},
				{
					Alias:  "API Key User",
					ApiKey: "this-is-a-valid-api-key-12345678901234567890",
					Capabilities: []types.Capability{
						{
							Actions: []string{"read:job", "write:job"},
						},
					},
				},
			},
		}

		// Execute
		authorizer, err := NewEntryPointAuthorizer(context.Background(), "test-node", authConfig)

		// Verify
		require.NoError(t, err)
		require.NotNil(t, authorizer)
	})

	t.Run("FailCreationWithNoUsers", func(t *testing.T) {
		// Setup
		authConfig := types.AuthConfig{
			Users: []types.AuthUser{}, // Empty users
		}

		// Execute
		authorizer, err := NewEntryPointAuthorizer(context.Background(), "test-node", authConfig)

		// Verify
		require.Error(t, err)
		assert.Nil(t, authorizer)
		assert.Contains(t, err.Error(), "no users configured")
	})

	t.Run("FailCreationWithInvalidUsers", func(t *testing.T) {
		// Setup
		authConfig := types.AuthConfig{
			Users: []types.AuthUser{
				{
					Alias:    "Valid User",
					Username: "validuser",
					Password: "password1234567",
					Capabilities: []types.Capability{
						{
							Actions: []string{"read:job"},
						},
					},
				},
				{
					Alias: "Invalid User", // Missing auth method
					Capabilities: []types.Capability{
						{
							Actions: []string{"read:job"},
						},
					},
				},
			},
		}

		// Execute
		authorizer, err := NewEntryPointAuthorizer(context.Background(), "test-node", authConfig)

		// Verify
		require.Error(t, err)
		assert.Nil(t, authorizer)
		assert.Contains(t, err.Error(), "has neither username/password nor API key")
	})
}

// Helper functions to create test data

func createTestEntryPointAuthorizer() *entryPointAuthorizer {
	// Create basic auth users map
	basicAuthUsers := map[string]types.AuthUser{
		"testuser": {
			Alias:    "Test User",
			Username: "testuser",
			Password: "testpass",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job"},
				},
			},
		},
	}

	// Create API key users map
	apiKeyUsers := map[string]types.AuthUser{
		"valid-api-key-123": {
			Alias:  "API User",
			ApiKey: "valid-api-key-123",
			Capabilities: []types.Capability{
				{
					Actions: []string{"read:job", "write:job"},
				},
			},
		},
	}

	// Create capability checker
	capabilityChecker := NewCapabilityChecker()

	// Create endpoint permissions (using the default ones)
	endpointPermissions := GetDefaultEndpointPermissions()

	// Add a custom endpoint for testing
	endpointPermissions["/v1/jobs"] = "job"
	endpointPermissions["/v1/admin"] = "node"

	// Create the basic auth authorizer
	basicAuthAuthorizer := NewBasicAuthAuthorizer(
		"test-node",
		basicAuthUsers,
		capabilityChecker,
		endpointPermissions,
	)

	// Create the API key authorizer
	apiKeyAuthorizer := NewApiKeyAuthorizer(
		"test-node",
		apiKeyUsers,
		capabilityChecker,
		endpointPermissions,
	)

	// Return the entry point authorizer
	return &entryPointAuthorizer{
		nodeID:              "test-node",
		basicAuthUsers:      basicAuthUsers,
		apiKeyUsers:         apiKeyUsers,
		capabilityChecker:   capabilityChecker,
		endpointPermissions: endpointPermissions,
		basicAuthAuthorizer: basicAuthAuthorizer,
		apiKeyAuthorizer:    apiKeyAuthorizer,
	}
}

func createTestEntryPointRequest(path, authHeader string) *http.Request {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:1234%s", path), nil)
	req.URL, _ = url.Parse(fmt.Sprintf("http://localhost:1234%s", path))

	if authHeader != "" {
		req.Header = http.Header{}
		req.Header.Add("Authorization", authHeader)
	}

	return req
}

// Helper function to create a mock JWT token for testing
func createMockJWTToken() string {
	// Create a mock header
	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Create a mock payload
	payload := map[string]interface{}{
		"sub":         "test-user",
		"iat":         1516239022,
		"exp":         9999999999, // Far future
		"permissions": []string{"read:job"},
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Create a mock signature (doesn't need to be valid for detection tests)
	signatureEncoded := "mock_signature_not_real_just_for_testing_jwt_detection"

	// Combine parts
	return headerEncoded + "." + payloadEncoded + "." + signatureEncoded
}

// TestIsJWTToken tests the JWT token detection function
func TestIsJWTToken(t *testing.T) {
	t.Run("ValidJWTToken", func(t *testing.T) {
		// Create a valid JWT token format
		token := createMockJWTToken()

		// Test detection
		result := isJWTToken(token)

		// Should be detected as a JWT
		assert.True(t, result, "Valid JWT token should be detected")
	})

	t.Run("ApiKeyNotDetectedAsJWT", func(t *testing.T) {
		// An API key with no dots
		token := "api-key-without-dots-12345678901234567890"

		// Test detection
		result := isJWTToken(token)

		// Should not be detected as a JWT
		assert.False(t, result, "API key should not be detected as JWT")
	})

	t.Run("InvalidFormatNotDetectedAsJWT", func(t *testing.T) {
		// String with dots but not valid base64 or JSON
		token := "invalid.jwt.format"

		// Test detection
		result := isJWTToken(token)

		// Should not be detected as a JWT
		assert.False(t, result, "Invalid format should not be detected as JWT")
	})

	t.Run("CorrectSegmentsButInvalidHeaderNotDetectedAsJWT", func(t *testing.T) {
		// Create invalid header but valid structure
		invalidHeader := "not-valid-base64"
		validPayload := base64.RawURLEncoding.EncodeToString([]byte("{\"sub\":\"test\"}"))
		mockSignature := "signature123"

		token := invalidHeader + "." + validPayload + "." + mockSignature

		// Test detection
		result := isJWTToken(token)

		// Should not be detected as a JWT
		assert.False(t, result, "Token with invalid header should not be detected as JWT")
	})
}

// TestJWTRouting tests the JWT token routing in the entry point authorizer
func TestJWTRouting(t *testing.T) {
	// This test needs to create a new entryPointAuthorizer with mock JWT authorizer
	// that can signal it was called

	t.Run("JWTTokenRoutedToJWTAuthorizer", func(t *testing.T) {
		// Create a standard entry point authorizer
		standardAuthorizer := createTestEntryPointAuthorizer()

		// Create a testing JWT token
		jwtToken := createMockJWTToken()

		// Create a mock JWT authorizer that records it was called
		jwtAuthorizerCalled := false
		mockJWTAuthorizer := &mockAuthorizer{
			authorizeFunc: func(req *http.Request) (Authorization, error) {
				jwtAuthorizerCalled = true
				// Check that this was indeed called with a JWT token
				authHeader := req.Header.Get("Authorization")
				assert.True(t, strings.HasPrefix(authHeader, "Bearer "))
				assert.Equal(t, "Bearer "+jwtToken, authHeader)
				return Authorization{Approved: true, TokenValid: true}, nil
			},
		}

		// Replace the JWT authorizer in our test authorizer
		standardAuthorizer.jwtAuthorizer = mockJWTAuthorizer

		// Create request with JWT token
		req := createTestEntryPointRequest("/v1/jobs", "Bearer "+jwtToken)

		// Call authorize
		auth, err := standardAuthorizer.Authorize(req)

		// Verify results
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
		assert.True(t, jwtAuthorizerCalled, "JWT authorizer should have been called")
	})

	t.Run("ApiKeyRoutedToApiKeyAuthorizer", func(t *testing.T) {
		// Create a standard entry point authorizer
		standardAuthorizer := createTestEntryPointAuthorizer()

		// API key (not a JWT token)
		apiKey := "valid-api-key-123"

		// Create a mock API key authorizer that records it was called
		apiKeyAuthorizerCalled := false
		mockApiKeyAuthorizer := &mockAuthorizer{
			authorizeFunc: func(req *http.Request) (Authorization, error) {
				apiKeyAuthorizerCalled = true
				// Check that this was indeed called with an API key
				authHeader := req.Header.Get("Authorization")
				assert.True(t, strings.HasPrefix(authHeader, "Bearer "))
				assert.Equal(t, "Bearer "+apiKey, authHeader)
				return Authorization{Approved: true, TokenValid: true}, nil
			},
		}

		// Replace the API key authorizer in our test authorizer
		standardAuthorizer.apiKeyAuthorizer = mockApiKeyAuthorizer

		// Also add a mock JWT authorizer to ensure it doesn't get called
		jwtAuthorizerCalled := false
		mockJWTAuthorizer := &mockAuthorizer{
			authorizeFunc: func(req *http.Request) (Authorization, error) {
				jwtAuthorizerCalled = true
				return Authorization{}, fmt.Errorf("JWT authorizer should not be called")
			},
		}
		standardAuthorizer.jwtAuthorizer = mockJWTAuthorizer

		// Create request with API key
		req := createTestEntryPointRequest("/v1/jobs", "Bearer "+apiKey)

		// Call authorize
		auth, err := standardAuthorizer.Authorize(req)

		// Verify results
		require.NoError(t, err)
		assert.True(t, auth.Approved)
		assert.True(t, auth.TokenValid)
		assert.True(t, apiKeyAuthorizerCalled, "API key authorizer should have been called")
		assert.False(t, jwtAuthorizerCalled, "JWT authorizer should not have been called")
	})
}

// mockAuthorizer is a simple implementation of the Authorizer interface for testing
type mockAuthorizer struct {
	authorizeFunc func(req *http.Request) (Authorization, error)
}

func (m *mockAuthorizer) Authorize(req *http.Request) (Authorization, error) {
	return m.authorizeFunc(req)
}
