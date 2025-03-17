package sso

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// DeviceCodeResponse represents the response from the device authorization endpoint
// This is our own representation of the device code response
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	Message                 string `json:"message"`
}

// OAuth2Service handles OAuth2 authentication flows
type OAuth2Service struct {
	config      types.Oauth2Config
	oauthConfig *oauth2.Config
}

// NewOAuth2Service creates a new OAuth2Service with the provided configuration
func NewOAuth2Service(nodeOauth2Config types.Oauth2Config) *OAuth2Service {
	// Create oauth2.Config with the appropriate settings
	oauthConfig := &oauth2.Config{
		ClientID: nodeOauth2Config.DeviceClientId,
		Scopes:   nodeOauth2Config.Scopes,
		Endpoint: oauth2.Endpoint{
			TokenURL:      nodeOauth2Config.TokenEndpoint,
			DeviceAuthURL: nodeOauth2Config.DeviceAuthorizationEndpoint,
			AuthStyle:     oauth2.AuthStyleInParams, // Device flow typically uses params
		},
	}

	return &OAuth2Service{
		config:      nodeOauth2Config,
		oauthConfig: oauthConfig,
	}
}

// InitiateDeviceCodeFlow starts the device code flow and returns the device code response
func (s *OAuth2Service) InitiateDeviceCodeFlow(ctx context.Context) (*DeviceCodeResponse, error) {
	// Use built-in DeviceAuth method from oauth2 library
	var authOptions []oauth2.AuthCodeOption

	// Add audience if specified
	if s.config.Audience != "" {
		authOptions = append(authOptions, oauth2.SetAuthURLParam("audience", s.config.Audience))
	}

	// Call the library's DeviceAuth method
	deviceResp, err := s.oauthConfig.DeviceAuth(ctx, authOptions...)
	if err != nil {
		return nil, fmt.Errorf("device authorization request failed: %w", err)
	}

	// Convert the library's response to our response type
	response := &DeviceCodeResponse{
		DeviceCode:              deviceResp.DeviceCode,
		UserCode:                deviceResp.UserCode,
		VerificationURI:         deviceResp.VerificationURI,
		VerificationURIComplete: deviceResp.VerificationURIComplete,
		ExpiresIn:               int(deviceResp.Expiry.Sub(time.Now()).Seconds()),
		Interval:                int(deviceResp.Interval),
	}

	return response, nil
}

// PollForToken polls the token endpoint until a token is granted or the context is canceled
func (s *OAuth2Service) PollForToken(ctx context.Context, deviceCode string) (*oauth2.Token, error) {
	// Create a DeviceAuthResponse to pass to DeviceAccessToken
	deviceResp := &oauth2.DeviceAuthResponse{
		DeviceCode: deviceCode,
		// Other fields aren't needed for the token exchange
	}

	// Use built-in DeviceAccessToken method to handle the polling
	token, err := s.oauthConfig.DeviceAccessToken(ctx, deviceResp)
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) {
			return nil, fmt.Errorf("token retrieval failed: %s - %s",
				retrieveErr.ErrorCode, retrieveErr.ErrorDescription)
		}
		return nil, fmt.Errorf("failed to obtain token: %w", err)
	}

	return token, nil
}
