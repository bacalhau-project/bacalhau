package authz

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/credsecurity"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/pkg/errors"
)

type basicAuthAuthorizer struct {
	nodeID string
	// Separate maps for different authentication methods
	basicAuthUsers map[string]types.AuthUser // Key is username
	// Capability checker for authorization
	capabilityChecker *CapabilityChecker
	// Endpoint permissions mapping
	endpointPermissions map[string]string
	// Bcrypt manager for password verification
	bcryptManager *credsecurity.BcryptManager
}

// validateBasicAuth validates basic authentication credentials
func (a *basicAuthAuthorizer) validateBasicAuth(authHeader string) (types.AuthUser, bool, error) {
	// Extract and decode the credentials
	encodedCredentials := authHeader[6:] // Skip "Basic "
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		return types.AuthUser{}, false, errors.Wrap(err, "failed to decode basic auth credentials")
	}

	// Split the credentials into username and password
	credentials := string(decodedBytes)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return types.AuthUser{}, false, errors.New("invalid basic auth credentials format")
	}

	username, password := parts[0], parts[1]

	// Look up the user by username (case-insensitive)
	user, exists := a.basicAuthUsers[strings.ToLower(username)]
	if !exists {
		return types.AuthUser{}, false, errors.New("invalid basic auth credentials")
	}

	// Check if the stored password is a bcrypt hash
	if a.bcryptManager.IsBcryptHash(user.Password) {
		// Verify using bcrypt
		err = a.bcryptManager.VerifyPassword(password, user.Password)
		if err != nil {
			return types.AuthUser{}, false, errors.New("invalid basic auth credentials")
		}
	} else {
		// Compare passwords using constant-time comparison for plain text
		if subtle.ConstantTimeCompare([]byte(user.Password), []byte(password)) != 1 {
			return types.AuthUser{}, false, errors.New("invalid basic auth credentials")
		}
	}

	// Authentication successful
	return user, true, nil
}

func NewBasicAuthAuthorizer(
	nodeID string,
	basicAuthUsers map[string]types.AuthUser,
	capabilityChecker *CapabilityChecker,
	endpointPermissions map[string]string,
) Authorizer {
	authorizer := &basicAuthAuthorizer{
		nodeID:              nodeID,
		basicAuthUsers:      basicAuthUsers,
		capabilityChecker:   capabilityChecker,
		endpointPermissions: endpointPermissions,
		bcryptManager:       credsecurity.NewDefaultBcryptManager(),
	}

	return authorizer
}

// Authorize implements the Authorizer interface
func (a *basicAuthAuthorizer) Authorize(req *http.Request) (Authorization, error) {
	if req.URL == nil {
		return Authorization{
			Approved:   false,
			TokenValid: false,
		}, apimodels.NewUnauthorizedError("unauthorized: missing URL")
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
		return Authorization{}, apimodels.NewUnauthorizedError("missing authorization header")
	}

	authHeader := authorizationHeaders[0]
	user, authenticated, authErr := a.validateBasicAuth(authHeader)

	// Handle authentication error
	if authErr != nil {
		return Authorization{
			Approved:   false,
			TokenValid: false,
		}, apimodels.NewUnauthorizedError(fmt.Sprintf("authentication failed: %s", authErr.Error()))
	}

	// Check if authentication succeeded
	if authenticated {
		// Check user capabilities using the capability checker
		hasCapability, requiredCapability, err := a.capabilityChecker.CheckUserAccess(user, resourceType, req)
		if err != nil {
			return Authorization{
				Approved:   false,
				TokenValid: true,
			}, err
		}

		if hasCapability {
			return Authorization{
				Approved:   true,
				TokenValid: true,
			}, nil
		} else {
			return Authorization{
					Approved:   false,
					TokenValid: true,
				}, apimodels.NewUnauthorizedError(fmt.Sprintf("user '%s' does not have the required capability '%s'",
					user.Alias, requiredCapability))
		}
	}

	// If we get here, something unexpected happened
	return Authorization{
		Approved:   false,
		TokenValid: false,
	}, apimodels.NewUnauthorizedError("authentication failed")
}
