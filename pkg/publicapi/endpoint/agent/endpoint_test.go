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
	"github.com/bacalhau-project/bacalhau/pkg/licensing"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

// cSpell:disable
const validOfficialTestLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6ImU2NmQxZjNhLWE4ZDgtNGQ1Ny04ZjE0LTAwNzIyODQ0YWZlMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6e30sImlhdCI6MTczNjg4MTYzOCwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoyMzg0ODgxNjM4LCJqdGkiOiJlNjZkMWYzYS1hOGQ4LTRkNTctOGYxNC0wMDcyMjg0NGFmZTIifQ.U6qkWmki2wp3RbPdn8d0zzsy4FchZIyUDmJi2bJ4w4vhwJlJ0_F2_317v4iPzy9q69eJOKNaqj8P3xYaPbpiooFm15OdJ3ecbMy8bKvvWVj43stw6HNP_uoW-RlZnY2zTOQ9WhlOhjnUPPC-UXOcaMwxiLBwMo5n3Rs0W9uAQHGQIptGg0sKiZvIrMZZ3vww2PZ3wJDiDvznE2lPtI7jAbcFFKDlhY3UiXed2ihGTWvLW8Zwj4veCR4PAUoEDu-nfQDvlqNeAvABT-KrKY2M-d5T_WzK1WwXtHok9tG2OV5ybSZoxFDQW3iqiCg6TqMwCAa6C6MBXtLnv-NP1H9Ytg"
const officialTokenButExpired = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6IjBkZDA0Yzg0LTA5YjgtNDE3OS04OGY3LWM3MmE5ZDU2YzBhMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6eyJzb21lTWV0YWRhdGEiOiJ2YWx1ZU9mU29tZU1ldGFkYXRhIn0sImlhdCI6MTczNjg5MTEzMSwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoxNzM2MjQxMDk4LCJqdGkiOiIwZGQwNGM4NC0wOWI4LTQxNzktODhmNy1jNzJhOWQ1NmMwYTIifQ.URD1ofoJwrleEkXWQ7vWVv_gCzwM-1cR6_6SOIf-d7Uuh3ttFJdNMDw_gbZp65sgLMycQKkm5ngooxK-FSwVj6jl2c70SvzuEHbdUsSZClLReOSbmY7CO6bOQYzQYVEeoWiykVMdgj2REgnrP3b2n4KGyTFKoqqXYpdjSJ9BXXgw-RfkXmyBV1h8imymcXCZcYxzcKPSDSoZLUrPSqD5ooM021VKaTd4J4jFql3BrLGrvaRgUtSgfQdJjo1alMUalZ7hAEWkmhBlQ_ocdlHeJOR3Rrlk5c-JANOJ4UslMLG465QJ8tmfxaUbbOPB2YPj0f9uEbGW5kGkHW3BKQZbDQ"

// TestEndpointConfigRedactFields asserts that auth tokens in the config are redacted.
func TestEndpointConfigRedactFields(t *testing.T) {
	router := echo.New()

	// Create license manager
	licenseManager, err := licensing.NewLicenseManager(&types.License{LocalPath: ""})
	require.NoError(t, err)

	// populate the fields that should be redacted with "secret" values.
	_, err = NewEndpoint(EndpointParams{
		Router:         router,
		LicenseManager: licenseManager,
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
	assert.Equal(t, payload.Config.Orchestrator.Auth.Token, "<redacted>")
	assert.Equal(t, payload.Config.Compute.Auth.Token, "<redacted>")
}

// TestEndpointLicenseValid tests the license endpoint when a valid license is configured
func TestEndpointLicenseValid(t *testing.T) {
	router := echo.New()

	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Write valid license file
	licenseContent := fmt.Sprintf(`{
		"license": %q
	}`, validOfficialTestLicense)
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	// Create license manager
	licenseManager, err := licensing.NewLicenseManager(&config.Orchestrator.License)
	require.NoError(t, err)

	_, err = NewEndpoint(EndpointParams{
		Router:         router,
		BacalhauConfig: config,
		LicenseManager: licenseManager,
	})
	require.NoError(t, err)

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
	assert.False(t, response.IsExpired())
	assert.Equal(t, 1, response.MaxNumberOfNodes())
}

// TestEndpointLicenseNotConfigured tests the license endpoint when no license is configured
func TestEndpointLicenseNotConfigured(t *testing.T) {
	router := echo.New()

	config := types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				// No license path configured
			},
		},
	}

	// Create license manager
	licenseManager, err := licensing.NewLicenseManager(&config.Orchestrator.License)
	require.NoError(t, err)

	_, err = NewEndpoint(EndpointParams{
		Router:         router,
		BacalhauConfig: config,
		LicenseManager: licenseManager,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)

	var errResp map[string]string
	err = json.NewDecoder(rr.Body).Decode(&errResp)
	require.NoError(t, err)
	require.Contains(t, errResp["message"], "Error inspecting orchestrator license: No license configured for orchestrator.")
}

// TestEndpointLicenseManagerNotConfigured tests when the license manager is not configured
func TestEndpointLicenseManagerNotConfigured(t *testing.T) {
	router := echo.New()

	_, err := NewEndpoint(EndpointParams{
		Router: router,
		// No license manager configured
	})
	require.ErrorContains(
		t,
		err,
		"license manager is required for agent endpoint",
	)
}

// TestEndpointLicenseExpired tests when the license token is valid but expired
func TestEndpointLicenseExpired(t *testing.T) {
	router := echo.New()

	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Write valid but expired license file
	licenseContent := fmt.Sprintf(`{
		"license": %q
	}`, officialTokenButExpired)
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	// Create license manager
	licenseManager, err := licensing.NewLicenseManager(&config.Orchestrator.License)
	require.NoError(t, err)

	_, err = NewEndpoint(EndpointParams{
		Router:         router,
		BacalhauConfig: config,
		LicenseManager: licenseManager,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/license", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var response apimodels.GetAgentLicenseResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	// Verify some basic claims
	assert.Equal(t, "Bacalhau", response.Product)
	assert.Equal(t, "0dd04c84-09b8-4179-88f7-c72a9d56c0a2", response.LicenseID)
	assert.Equal(t, "standard", response.LicenseType)
	assert.Equal(t, "test-customer-id-123", response.CustomerID)
	assert.Equal(t, "v1", response.LicenseVersion)
	assert.Equal(t, "1", response.Capabilities["max_nodes"])
	assert.Equal(t, "https://expanso.io/", response.Issuer)
	assert.Equal(t, "test-customer-id-123", response.Subject)
	assert.Equal(t, "0dd04c84-09b8-4179-88f7-c72a9d56c0a2", response.ID)
	assert.Equal(t, int64(1736241098), response.ExpiresAt.Unix())
	assert.True(t, response.IsExpired())
	assert.Equal(t, 1, response.MaxNumberOfNodes())
}
