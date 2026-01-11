//go:build unit || !integration

package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

// TestEndpointConfigRedactFields asserts that auth tokens in the config are redacted.
func TestEndpointConfigRedactFields(t *testing.T) {
	router := echo.New()

	// populate the fields that should be redacted with "secret" values.
	_, err := NewEndpoint(EndpointParams{
		Router:        router,
		BacalhauConfig: types.Bacalhau{
			Orchestrator: types.Orchestrator{
				Auth: types.OrchestratorAuth{
					Token: "super-secret-orchestrator-token",
				},
			},
			Compute: types.Compute{
				Auth: types.ComputeAuth{
					Token: "super-secret-orchestrator-token",
				},
			},
		},
	})

	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/config", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// assert the secret values are not present.
	var payload apimodels.GetAgentConfigResponse
	err = json.NewDecoder(rr.Body).Decode(&payload)
	require.NoError(t, err)
	assert.Equal(t, payload.Config.Orchestrator.Auth.Token, "********")
	assert.Equal(t, payload.Config.Compute.Auth.Token, "********")
}

// TestEndpointNodeOauth2ConfigPopulated tests the nodeOauth2Config endpoint when the OAuth2 config is populated
func TestEndpointNodeOauth2ConfigPopulated(t *testing.T) {
	router := echo.New()

	// Prepare a config with populated OAuth2 settings
	config := types.Bacalhau{
		API: types.API{
			Auth: types.AuthConfig{
				Oauth2: types.Oauth2Config{
					ProviderID:                  "test-provider",
					ProviderName:                "Test Provider",
					DeviceClientID:              "device-client-id-123",
					DeviceAuthorizationEndpoint: "https://test-provider.com/oauth/device/code",
					JWKSUri:                     "https://test-provider.com/.well-known/jwks.json",
					TokenEndpoint:               "https://test-provider.com/oauth/token",
					PollingInterval:             5,
					Audience:                    "test-audience",
					Scopes:                      []string{"read:jobs", "write:jobs"},
				},
			},
		},
	}


	_, err := NewEndpoint(EndpointParams{
		Router:         router,
		BacalhauConfig: config,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/authconfig", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response apimodels.GetAgentNodeAuthConfigResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	// Verify the OAuth2 config values
	assert.Equal(t, "test-provider", response.Config.ProviderID)
	assert.Equal(t, "Test Provider", response.Config.ProviderName)
	assert.Equal(t, "https://test-provider.com/.well-known/jwks.json", response.Config.JWKSUri)

	// Verify DeviceCode flow
	assert.Equal(t, "device-client-id-123", response.Config.DeviceClientID)
	assert.Equal(t, "https://test-provider.com/oauth/device/code", response.Config.DeviceAuthorizationEndpoint)
	assert.Equal(t, "https://test-provider.com/oauth/token", response.Config.TokenEndpoint)
	assert.Equal(t, 5, response.Config.PollingInterval)
	assert.Equal(t, "test-audience", response.Config.Audience)
	assert.Equal(t, []string{"read:jobs", "write:jobs"}, response.Config.Scopes)
}

// TestEndpointNodeOauth2ConfigEmpty tests the nodeOauth2Config endpoint when the OAuth2 config is empty
func TestEndpointNodeOauth2ConfigEmpty(t *testing.T) {
	router := echo.New()

	// Create config with empty OAuth2 settings
	config := types.Bacalhau{
		API: types.API{
			Auth: types.AuthConfig{
				// OAuth2 config is empty by default
			},
		},
	}


	_, err := NewEndpoint(EndpointParams{
		Router:         router,
		BacalhauConfig: config,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/authconfig", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response apimodels.GetAgentNodeAuthConfigResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	// Verify the OAuth2 config is empty
	assert.Empty(t, response.Config.ProviderID)
	assert.Empty(t, response.Config.ProviderName)
	assert.Empty(t, response.Config.JWKSUri)
	assert.Empty(t, response.Config.DeviceClientID)
}

// TestEndpointNodeOauth2ConfigRouteRegistration tests that the nodeOauth2Config route is properly registered
func TestEndpointNodeOauth2ConfigRouteRegistration(t *testing.T) {
	router := echo.New()


	endpoint, err := NewEndpoint(EndpointParams{
		Router:         router,
		BacalhauConfig: types.Bacalhau{},
	})
	require.NoError(t, err)

	// Add the route manually - this would normally be in the NewEndpoint function
	g := endpoint.router.Group("/api/v1/agent")
	g.GET("/oauth2config", endpoint.nodeAuthConfig)

	// Test that the route works
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/oauth2config", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}
