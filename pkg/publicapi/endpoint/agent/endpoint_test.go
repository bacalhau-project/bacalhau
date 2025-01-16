//go:build unit || !integration

package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

const validOfficialTestLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6ImU2NmQxZjNhLWE4ZDgtNGQ1Ny04ZjE0LTAwNzIyODQ0YWZlMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6e30sImlhdCI6MTczNjg4MTYzOCwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoyMzg0ODgxNjM4LCJqdGkiOiJlNjZkMWYzYS1hOGQ4LTRkNTctOGYxNC0wMDcyMjg0NGFmZTIifQ.U6qkWmki2wp3RbPdn8d0zzsy4FchZIyUDmJi2bJ4w4vhwJlJ0_F2_317v4iPzy9q69eJOKNaqj8P3xYaPbpiooFm15OdJ3ecbMy8bKvvWVj43stw6HNP_uoW-RlZnY2zTOQ9WhlOhjnUPPC-UXOcaMwxiLBwMo5n3Rs0W9uAQHGQIptGg0sKiZvIrMZZ3vww2PZ3wJDiDvznE2lPtI7jAbcFFKDlhY3UiXed2ihGTWvLW8Zwj4veCR4PAUoEDu-nfQDvlqNeAvABT-KrKY2M-d5T_WzK1WwXtHok9tG2OV5ybSZoxFDQW3iqiCg6TqMwCAa6C6MBXtLnv-NP1H9Ytg"

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

// TestEndpointLicense tests the license endpoint
func TestEndpointLicense(t *testing.T) {
	// Create a temporary license file
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Create license file with proper JSON format
	licenseContent := fmt.Sprintf(`{
		"license": %q
	}`, validOfficialTestLicense)

	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	router := echo.New()

	NewEndpoint(EndpointParams{
		Router: router,
		BacalhauConfig: types.Bacalhau{
			Orchestrator: types.Orchestrator{
				License: types.License{
					LocalPath: licensePath,
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response apimodels.GetAgentLicenseResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	// Verify some basic claims
	assert.Equal(t, "Bacalhau", response.Product)
	assert.Equal(t, "e66d1f3a-a8d8-4d57-8f14-00722844afe2", response.LicenseID)
	assert.Equal(t, "standard", response.LicenseType)
	assert.Equal(t, "test-customer-id-123", response.CustomerID)
	assert.Equal(t, "v1", response.LicenseVersion)
	assert.Equal(t, "1", response.Capabilities["max_nodes"])
	assert.Equal(t, "https://expanso.io/", response.Issuer)
	assert.Equal(t, "test-customer-id-123", response.Subject)
	assert.Equal(t, "e66d1f3a-a8d8-4d57-8f14-00722844afe2", response.ID)
	assert.Equal(t, int64(2384881638), response.ExpiresAt.Unix())
}

// TestEndpointLicenseNoLicense tests the license endpoint when no license is configured
func TestEndpointLicenseNoLicense(t *testing.T) {
	router := echo.New()

	NewEndpoint(EndpointParams{
		Router: router,
		BacalhauConfig: types.Bacalhau{
			Orchestrator: types.Orchestrator{
				License: types.License{
					// No license path configured
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)

	// Also verify the error message
	var errResp map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, "Node license not configured", errResp["message"])
}

// TestEndpointLicenseFileNotFound tests when the license file doesn't exist
func TestEndpointLicenseFileNotFound(t *testing.T) {
	router := echo.New()

	NewEndpoint(EndpointParams{
		Router: router,
		BacalhauConfig: types.Bacalhau{
			Orchestrator: types.Orchestrator{
				License: types.License{
					LocalPath: "/non/existent/path/license.json",
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var errResp map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp["message"].(string), "failed to read license file")
}

// TestEndpointLicenseInvalidJSON tests when the license file contains invalid JSON
func TestEndpointLicenseInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Write invalid JSON to the license file
	err := os.WriteFile(licensePath, []byte("invalid json content"), 0644)
	require.NoError(t, err)

	router := echo.New()

	NewEndpoint(EndpointParams{
		Router: router,
		BacalhauConfig: types.Bacalhau{
			Orchestrator: types.Orchestrator{
				License: types.License{
					LocalPath: licensePath,
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var errResp map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp["message"].(string), "failed to parse license file")
}

// TestEndpointLicenseInvalidLicenseFormat tests when the license file is valid JSON but missing the license field
func TestEndpointLicenseInvalidLicenseFormat(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Write JSON without the required "license" field
	err := os.WriteFile(licensePath, []byte(`{"some_other_field": "value"}`), 0644)
	require.NoError(t, err)

	router := echo.New()

	NewEndpoint(EndpointParams{
		Router: router,
		BacalhauConfig: types.Bacalhau{
			Orchestrator: types.Orchestrator{
				License: types.License{
					LocalPath: licensePath,
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var errResp map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp["message"].(string), "failed to validate license")
}

// TestEndpointLicenseInvalidToken tests when the license token is invalid
func TestEndpointLicenseInvalidToken(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Write JSON with an invalid JWT token
	licenseContent := fmt.Sprintf(`{
		"license": "invalid.jwt.token"
	}`)
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	router := echo.New()

	NewEndpoint(EndpointParams{
		Router: router,
		BacalhauConfig: types.Bacalhau{
			Orchestrator: types.Orchestrator{
				License: types.License{
					LocalPath: licensePath,
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var errResp map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp["message"].(string), "failed to validate license: failed to parse license: token is malformed")
}
