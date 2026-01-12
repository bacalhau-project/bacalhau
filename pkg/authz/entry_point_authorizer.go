package authz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

const (
	MinimumPasswordLength = 10
	MaximumPasswordLength = 255
	MinimumAPIKeyLength   = 20
	MaximumAPIKeyLength   = 255
	UsernameMaxLength     = 100
)

type entryPointAuthorizer struct {
	nodeID string
	// Separate maps for different authentication methods
	basicAuthUsers map[string]types.AuthUser // Key is username
	apiKeyUsers    map[string]types.AuthUser // Key is API key
	// Capability checker for authorization
	capabilityChecker *CapabilityChecker
	// Endpoint permissions mapping
	endpointPermissions map[string]string

	// Specialized Authorizers
	basicAuthAuthorizer Authorizer
	apiKeyAuthorizer    Authorizer
	jwtAuthorizer       Authorizer
}

// validateUser validates a user configuration and returns an error if it's invalid
func (a *entryPointAuthorizer) validateUser(user types.AuthUser) error {
	// Get user identifier for error messages
	userID := getUserIdentifier(user)

	// Validation: Check that user has either username+password OR apiKey but not both
	hasUsernamePassword := user.Username != "" && user.Password != ""
	hasAPIKey := user.APIKey != ""

	if hasUsernamePassword && hasAPIKey {
		return fmt.Errorf("user '%s' has both username/password and API key, must have only one authentication method", userID)
	}

	if !hasUsernamePassword && !hasAPIKey {
		return fmt.Errorf("user '%s' has neither username/password nor API key, must have one authentication method", userID)
	}

	// Validate username format if provided
	if user.Username != "" {
		// Check username length
		if len(strings.TrimSpace(user.Username)) > UsernameMaxLength {
			return fmt.Errorf("username for user '%s' exceeds maximum length of %d characters", userID, UsernameMaxLength)
		}

		if len(strings.TrimSpace(user.Username)) == 0 {
			return fmt.Errorf("username for user '%s' should not be empty", userID)
		}

		// Check username is alphanumeric
		alphanumericRegex := regexp.MustCompile("^[a-zA-Z0-9]+$")
		if !alphanumericRegex.MatchString(user.Username) {
			return fmt.Errorf("username for user '%s' must contain only alphanumeric characters (a-z, A-Z, 0-9)", userID)
		}
	}

	// Validate password format if provided
	if user.Password != "" {
		// Check password length
		if len(user.Password) < MinimumPasswordLength {
			return fmt.Errorf("password for user '%s' is too short, minimum length is %d characters", userID, MinimumPasswordLength)
		}

		if len(user.Password) > MaximumPasswordLength {
			return fmt.Errorf("password for user '%s' exceeds maximum length of %d characters", userID, MaximumPasswordLength)
		}
	}

	// Validate API key format if provided
	if user.APIKey != "" {
		// Check API key length
		if len(user.APIKey) < MinimumAPIKeyLength {
			return fmt.Errorf("API key for user '%s' is too short, minimum length is %d characters", userID, MinimumAPIKeyLength)
		}

		if len(user.APIKey) > MaximumAPIKeyLength {
			return fmt.Errorf("API key for user '%s' exceeds maximum length of %d characters", userID, MaximumAPIKeyLength)
		}
	}

	// Validation: Check that user has at least one capability
	if len(user.Capabilities) == 0 {
		return fmt.Errorf("user '%s' has no capabilities defined, must have at least one capability", userID)
	}

	return nil
}

// getUserIdentifier returns a string to identify the user in logs and error messages
// It prefers alias if present, then username if present, then last 5 chars of API key
func getUserIdentifier(user types.AuthUser) string {
	if user.Alias != "" {
		return user.Alias
	}
	if user.Username != "" {
		return user.Username
	}
	if user.APIKey != "" {
		// Get last 5 characters of API key
		const apiKeyMaskOffset = 5
		if len(user.APIKey) > apiKeyMaskOffset {
			return "API key ending in ..." + user.APIKey[len(user.APIKey)-5:]
		}
		return "API key " + user.APIKey
	}
	return "unknown user"
}

// checkForDuplicates checks for duplicate aliases, usernames, and API keys
// Parameters:
// - user: the current user being validated
// - seenAliases: map of lowercase aliases to their original casing
// - seenUsernames: map of lowercase usernames to their original casing
// - seenAPIKeys: map of API keys that have been seen
// Returns an error if any duplicates are found
func (a *entryPointAuthorizer) checkForDuplicates(
	user types.AuthUser,
	seenAliases map[string]string,
	seenUsernames map[string]string,
	seenAPIKeys map[string]bool,
) error {
	userID := getUserIdentifier(user)

	// Check for duplicate alias (case-insensitive) only if alias is defined
	if user.Alias != "" {
		aliasLower := strings.ToLower(user.Alias)
		if originalAlias, exists := seenAliases[aliasLower]; exists {
			return fmt.Errorf("duplicate alias detected: '%s' and '%s' (aliases are case-insensitive)", originalAlias, user.Alias)
		}
	}

	// Check for duplicate username (case-insensitive)
	if user.Username != "" {
		usernameLower := strings.ToLower(user.Username)
		if originalUsername, exists := seenUsernames[usernameLower]; exists {
			return fmt.Errorf("duplicate username detected: '%s' and '%s' (usernames are case-insensitive)",
				originalUsername, user.Username)
		}
	}

	// Check for duplicate API key
	if user.APIKey != "" && seenAPIKeys[user.APIKey] {
		return fmt.Errorf("duplicate API key detected for user '%s'", userID)
	}

	return nil
}

// validateAllUsers validates all users and checks for duplicates
func (a *entryPointAuthorizer) validateAllUsers(users []types.AuthUser) error {
	// Create maps to track used aliases, usernames, and API keys
	seenAliases := make(map[string]string)   // map[lowercase_alias]original_alias
	seenUsernames := make(map[string]string) // map[lowercase_username]original_username
	seenAPIKeys := make(map[string]bool)

	// Validate all users first
	for _, user := range users {
		// Validate the user
		if err := a.validateUser(user); err != nil {
			return err
		}

		// Check for duplicates
		if err := a.checkForDuplicates(user, seenAliases, seenUsernames, seenAPIKeys); err != nil {
			return err
		}

		// Update tracking maps after validation passes
		// Only track non-empty aliases
		if user.Alias != "" {
			aliasLower := strings.ToLower(user.Alias)
			seenAliases[aliasLower] = user.Alias
		}

		if user.Username != "" {
			usernameLower := strings.ToLower(user.Username)
			seenUsernames[usernameLower] = user.Username
		}

		if user.APIKey != "" {
			seenAPIKeys[user.APIKey] = true
		}
	}

	return nil
}

// populateUserMaps populates the basicAuthUsers and apiKeyUsers maps with valid users
func (a *entryPointAuthorizer) populateUserMaps(users []types.AuthUser) {
	for _, user := range users {
		if user.APIKey != "" {
			a.apiKeyUsers[user.APIKey] = user
		} else {
			// User must have username/password
			a.basicAuthUsers[strings.ToLower(user.Username)] = user
		}
	}
}

func NewEntryPointAuthorizer(ctx context.Context, nodeID string, authConfig types.AuthConfig) (Authorizer, error) {
	capabilityChecker := NewCapabilityChecker()
	endpointPermissions := GetDefaultEndpointPermissions()

	// Create the authorizer instance
	authorizer := &entryPointAuthorizer{
		nodeID:              nodeID,
		basicAuthUsers:      make(map[string]types.AuthUser),
		apiKeyUsers:         make(map[string]types.AuthUser),
		capabilityChecker:   capabilityChecker,
		endpointPermissions: endpointPermissions,
	}

	// Validate all users and check for duplicates
	if err := authorizer.validateAllUsers(authConfig.Users); err != nil {
		return nil, err
	}

	// Populate user maps with validated users
	authorizer.populateUserMaps(authConfig.Users)

	// Inject Child Authorizers
	authorizer.basicAuthAuthorizer = NewBasicAuthAuthorizer(
		nodeID,
		authorizer.basicAuthUsers,
		capabilityChecker,
		endpointPermissions,
	)

	authorizer.apiKeyAuthorizer = NewAPIKeyAuthorizer(
		nodeID,
		authorizer.apiKeyUsers,
		capabilityChecker,
		endpointPermissions,
	)

	createdJWTAuthorizer, err := NewJWTAuthorizer(
		ctx,
		nodeID,
		authConfig,
		capabilityChecker,
		endpointPermissions,
	)
	if err != nil {
		return nil, err
	}
	authorizer.jwtAuthorizer = createdJWTAuthorizer

	return authorizer, nil
}

// Authorize implements the Authorizer interface
func (a *entryPointAuthorizer) Authorize(req *http.Request) (Authorization, error) {
	if req.URL == nil {
		return Authorization{
			Approved:   false,
			TokenValid: false,
			Reason:     "Missing Request URL",
		}, nil
	}

	// Check if the endpoint is open (doesn't require authentication)
	reqPath := req.URL.Path
	resourceType := MapEndpointToResourceType(reqPath, a.endpointPermissions)

	// If endpoint is "open", approve without authentication
	if resourceType == ResourceTypeOpen {
		return Authorization{
			Approved:   true,
			TokenValid: true,
		}, nil
	}

	// Get Authorization header
	authorizationHeaders := req.Header["Authorization"]
	if len(authorizationHeaders) == 0 {
		return Authorization{
			Approved:   false,
			TokenValid: false,
			Reason:     "Missing Authorization header",
		}, nil
	}

	authHeader := authorizationHeaders[0]

	// Check the authentication method by inspecting the header prefix
	if strings.HasPrefix(authHeader, "Basic ") {
		// Validate Basic Auth
		return a.basicAuthAuthorizer.Authorize(req)
	} else if strings.HasPrefix(authHeader, "Bearer ") {
		// Extract the token
		token := authHeader[7:] // Skip "Bearer "

		// Use more sophisticated detection for JWT tokens
		if isJWTToken(token) {
			// If it looks like a JWT, use the JWT authorizer
			return a.jwtAuthorizer.Authorize(req)
		} else {
			// Otherwise, treat it as an API key
			return a.apiKeyAuthorizer.Authorize(req)
		}
	} else {
		// Unsupported authentication method
		return Authorization{
			Approved:   false,
			TokenValid: false,
			Reason:     "unsupported authentication method",
		}, nil
	}
}

// isJWTToken determines if a token is a JWT by checking its format and structure
// It performs a preliminary validation without validating the signature
func isJWTToken(tokenString string) bool {
	// Check if the token has the correct number of segments (3 segments = 2 periods)
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false
	}

	// Try to decode the header (first part)
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	// Check if the header is valid JSON
	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return false
	}

	// Try to decode the payload (second part)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	// Check if the payload is valid JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false
	}

	// If we got this far, it's likely a JWT token
	return true
}
