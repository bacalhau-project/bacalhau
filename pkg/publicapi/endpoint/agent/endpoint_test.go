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
	NewEndpoint(EndpointParams{
		Router: router,
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/config", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	// assert the secret values are not present.
	var payload apimodels.GetAgentConfigResponse
	err := json.NewDecoder(rr.Body).Decode(&payload)
	require.NoError(t, err)
	assert.Equal(t, payload.Config.Orchestrator.Auth.Token, "<redacted>")
	assert.Equal(t, payload.Config.Compute.Auth.Token, "<redacted>")
}
