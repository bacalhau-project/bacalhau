package authz

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// JWTClaims represents the expected claims in a JWT token
type JWTClaims struct {
	jwt.RegisteredClaims
	Permissions []string `json:"permissions,omitempty"`
}

type jwtAuthorizer struct {
	nodeID              string
	jwksURL             string
	keyFunc             jwt.Keyfunc
	capabilityChecker   *CapabilityChecker
	endpointPermissions map[string]string
	issuer              string
	audience            string
	deviceClientID      string
}

// validateOAuth2Config validates OAuth2 configuration and returns whether it's complete
// Returns:
// - isComplete: true if all required fields are present, false if all are empty or some are missing
// - error: nil if all fields are present or all are empty, error if some fields are present but others are missing
func validateOAuth2Config(oauth2Config types.Oauth2Config) (bool, error) {
	// Check if any of the fields are set
	hasJWKSUri := oauth2Config.JWKSUri != ""
	hasIssuer := oauth2Config.Issuer != ""
	hasAudience := oauth2Config.Audience != ""
	hasClientID := oauth2Config.DeviceClientID != ""

	// If any field is set, then all fields must be set (except PollingInterval)
	anyFieldSet := hasJWKSUri || hasIssuer || hasAudience || hasClientID

	// If none of the fields are set, return false for "config not complete" but no error
	if !anyFieldSet {
		return false, nil
	}

	// At this point, at least one field is set, so check if all required fields are set
	missingFields := []string{}

	if !hasJWKSUri {
		missingFields = append(missingFields, "JWKSUri")
	}
	if !hasIssuer {
		missingFields = append(missingFields, "Issuer")
	}
	if !hasAudience {
		missingFields = append(missingFields, "Audience")
	}
	if !hasClientID {
		missingFields = append(missingFields, "DeviceClientID")
	}

	if len(missingFields) > 0 {
		return false, fmt.Errorf("missing required OAuth2 fields for JWT authorization: %s", strings.Join(missingFields, ", "))
	}

	// Additional validation for JWKS URL format
	if hasJWKSUri {
		// Check if it's a valid URL
		if !isValidURL(oauth2Config.JWKSUri) {
			return false, fmt.Errorf("invalid JWKS URL format: %s", oauth2Config.JWKSUri)
		}
	}

	// All fields are present and valid
	return true, nil
}

// isValidURL checks if the provided string is a valid URL
func isValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// NewJWTAuthorizer creates a new JWT authorizer
func NewJWTAuthorizer(
	ctx context.Context,
	nodeID string,
	authConfig types.AuthConfig,
	capabilityChecker *CapabilityChecker,
	endpointPermissions map[string]string,
) (Authorizer, error) {
	// Validate OAuth2 configuration
	isComplete, err := validateOAuth2Config(authConfig.Oauth2)
	if err != nil {
		return nil, err
	}

	// If OAuth2 config is not complete, return DenyAuthorizer
	if !isComplete {
		return NewDenyAuthorizer("JWT authorization not configured"), nil
	}

	// Extract JWKS URL from auth config - using OAuth2 configuration
	jwksURL := authConfig.Oauth2.JWKSUri

	// Create the JWKS client with auto-refresh capability
	// We're using background context since this should run for the lifetime of the application
	jwks, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create JWKS client")
	}

	// Get the jwt.Keyfunc compatible function
	jwtKeyFunc := func(token *jwt.Token) (interface{}, error) {
		return jwks.Keyfunc(token)
	}

	// Extract other required configuration values
	issuer := authConfig.Oauth2.Issuer
	audience := authConfig.Oauth2.Audience
	clientID := authConfig.Oauth2.DeviceClientID

	authorizer := &jwtAuthorizer{
		nodeID:              nodeID,
		jwksURL:             jwksURL,
		keyFunc:             jwtKeyFunc,
		capabilityChecker:   capabilityChecker,
		endpointPermissions: endpointPermissions,
		issuer:              issuer,
		audience:            audience,
		deviceClientID:      clientID,
	}

	return authorizer, nil
}

// validateJWT validates the JWT token and extracts claims
func (a *jwtAuthorizer) validateJWT(tokenString string) (*JWTClaims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTClaims{},
		a.keyFunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(a.issuer),     // Validate issuer
		jwt.WithAudience(a.audience), // Validate audience
		jwt.WithExpirationRequired(), // Ensure token has an expiration
		jwt.WithIssuedAt(),           // Validate issued at time
	)

	if err != nil {
		return nil, errors.Wrap(err, "invalid JWT token")
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, errors.New("invalid JWT token")
	}

	// Extract the claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("invalid JWT claims")
	}

	return claims, nil
}

// Authorize implements the Authorizer interface
func (a *jwtAuthorizer) Authorize(req *http.Request) (Authorization, error) {
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

	// Extract the token from the Authorization header
	authHeader := authorizationHeaders[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return Authorization{
			Approved:   false,
			TokenValid: false,
		}, apimodels.NewUnauthorizedError("invalid authorization header format, expected 'Bearer TOKEN'")
	}

	// Get the token
	tokenString := authHeader[7:] // Skip "Bearer "

	// Validate the JWT token
	claims, err := a.validateJWT(tokenString)
	if err != nil {
		return Authorization{
			Approved:   false,
			TokenValid: false,
		}, apimodels.NewUnauthorizedError(fmt.Sprintf("JWT validation failed: %s", err.Error()))
	}

	// Create a virtual user from JWT claims
	user := types.AuthUser{
		Alias:    claims.Subject,
		Username: claims.Subject,
	}

	// Convert permissions from JWT to capabilities - put all permissions in one capability
	if len(claims.Permissions) > 0 {
		user.Capabilities = append(user.Capabilities, types.Capability{
			Actions: claims.Permissions,
		})
	}

	// Check if the user has the required capabilities for the requested resource
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

// DenyAuthorizer is a simple authorizer that always denies requests
type denyAuthorizer struct {
	reason string // Optional reason for denial
}

// NewDenyAuthorizer creates a new authorizer that always denies requests
func NewDenyAuthorizer(reason string) Authorizer {
	if reason == "" {
		reason = "access denied by policy"
	}
	return &denyAuthorizer{
		reason: reason,
	}
}

// Authorize implements the Authorizer interface but always returns unauthorized
func (a *denyAuthorizer) Authorize(req *http.Request) (Authorization, error) {
	return Authorization{
		Approved:   false,
		TokenValid: false,
	}, apimodels.NewUnauthorizedError(a.reason)
}
