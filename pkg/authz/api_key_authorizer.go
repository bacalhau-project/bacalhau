package authz

import (
	"fmt"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/pkg/errors"
)

type apiKeyAuthorizer struct {
	nodeID              string
	apiKeyUsers         map[string]types.AuthUser // Key is API key
	capabilityChecker   *CapabilityChecker
	endpointPermissions map[string]string
}

// validateAPIKey validates an API key from a Bearer token
func (a *apiKeyAuthorizer) validateAPIKey(authHeader string) (types.AuthUser, bool, error) {
	// Extract the API key
	apiKey := authHeader[7:] // Skip "Bearer "
	if apiKey == "" {
		return types.AuthUser{}, false, errors.New("empty API key provided")
	}

	// Look up the user by API key
	user, exists := a.apiKeyUsers[apiKey]
	if !exists {
		return types.AuthUser{}, false, errors.New("invalid API key")
	}

	// Authentication successful
	return user, true, nil
}

func NewAPIKeyAuthorizer(
	nodeID string,
	apiKeyUsers map[string]types.AuthUser,
	capabilityChecker *CapabilityChecker,
	endpointPermissions map[string]string,
) Authorizer {
	// Create the authorizer instance
	authorizer := &apiKeyAuthorizer{
		nodeID:              nodeID,
		apiKeyUsers:         apiKeyUsers,
		capabilityChecker:   capabilityChecker,
		endpointPermissions: endpointPermissions,
	}

	return authorizer
}

// Authorize implements the Authorizer interface
func (a *apiKeyAuthorizer) Authorize(req *http.Request) (Authorization, error) {
	if req.URL == nil {
		return Authorization{}, errors.New("bad HTTP request: missing URL")
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
	user, authenticated, authErr := a.validateAPIKey(authHeader)

	// Handle authentication error
	if authErr != nil {
		return Authorization{
			Approved:   false,
			TokenValid: false,
		}, apimodels.NewUnauthorizedError(fmt.Sprintf("authentication failed: %s", authErr.Error()))
	}

	if authenticated {
		hasCapability, requiredCapability, err := a.capabilityChecker.CheckUserAccess(user, resourceType, req)
		if err != nil {
			return Authorization{
				Approved:   false,
				TokenValid: true,
			}, err
		}

		if hasCapability {
			return Authorization{Approved: true, TokenValid: true}, nil
		} else {
			return Authorization{Approved: false, TokenValid: true},
				apimodels.NewUnauthorizedError(
					fmt.Sprintf(
						"user '%s' does not have the required capability '%s'",
						user.Alias, requiredCapability),
				)
		}
	}

	// If we get here, something unexpected happened
	return Authorization{
		Approved:   false,
		TokenValid: false,
	}, apimodels.NewUnauthorizedError("authentication failed")
}
