package licensing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/stretchr/testify/require"
)

// cSpell:disable
const validOfficialTestLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVuSm5GQ05TeUFUMVNRdnR6bDc4MllDZUdrV3FUQ3R2MWZ5SFVRa3hyTlUiLCJ0eXAiOiJKV1QifQ.eyJwcm9kdWN0IjoiQmFjYWxoYXUiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsImxpY2Vuc2VfdHlwZSI6InN0YW5kYXJkIiwibGljZW5zZV9pZCI6ImU2NmQxZjNhLWE4ZDgtNGQ1Ny04ZjE0LTAwNzIyODQ0YWZlMiIsImN1c3RvbWVyX25hbWUiOiJiYWNhbGhhdS1pbnRlZ3JhdGlvbi10ZXN0cyIsImN1c3RvbWVyX2lkIjoidGVzdC1jdXN0b21lci1pZC0xMjMiLCJjYXBhYmlsaXRpZXMiOnsibWF4X25vZGVzIjoiMSJ9LCJtZXRhZGF0YSI6e30sImlhdCI6MTczNjg4MTYzOCwiaXNzIjoiaHR0cHM6Ly9leHBhbnNvLmlvLyIsInN1YiI6InRlc3QtY3VzdG9tZXItaWQtMTIzIiwiZXhwIjoyMzg0ODgxNjM4LCJqdGkiOiJlNjZkMWYzYS1hOGQ4LTRkNTctOGYxNC0wMDcyMjg0NGFmZTIifQ.U6qkWmki2wp3RbPdn8d0zzsy4FchZIyUDmJi2bJ4w4vhwJlJ0_F2_317v4iPzy9q69eJOKNaqj8P3xYaPbpiooFm15OdJ3ecbMy8bKvvWVj43stw6HNP_uoW-RlZnY2zTOQ9WhlOhjnUPPC-UXOcaMwxiLBwMo5n3Rs0W9uAQHGQIptGg0sKiZvIrMZZ3vww2PZ3wJDiDvznE2lPtI7jAbcFFKDlhY3UiXed2ihGTWvLW8Zwj4veCR4PAUoEDu-nfQDvlqNeAvABT-KrKY2M-d5T_WzK1WwXtHok9tG2OV5ybSZoxFDQW3iqiCg6TqMwCAa6C6MBXtLnv-NP1H9Ytg"

func TestLicenseManager_ValidateLicense_InvalidToken(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	licenseContent := `{"license": "invalid-token"}`
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	claims, err := manager.ValidateLicense()
	require.Error(t, err)
	require.Nil(t, claims)
	require.Contains(t, err.Error(), "invalid license: failed to parse license: token is malformed")
}

func TestLicenseManager_ValidateLicense_ValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	licenseContent := fmt.Sprintf(`{"license": %q}`, validOfficialTestLicense)
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	claims, err := manager.ValidateLicense()
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Verify some basic claims
	require.Equal(t, "Bacalhau", claims.Product)
	require.Equal(t, "e66d1f3a-a8d8-4d57-8f14-00722844afe2", claims.LicenseID)
	require.Equal(t, "standard", claims.LicenseType)
	require.Equal(t, "test-customer-id-123", claims.CustomerID)
	require.Equal(t, "v1", claims.LicenseVersion)
	require.Equal(t, "1", claims.Capabilities["max_nodes"])
}

func TestLicenseManager_ValidateLicense_NoLicenseConfigured(t *testing.T) {
	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				// No license path configured
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	claims, err := manager.ValidateLicense()
	require.Error(t, err)
	require.Nil(t, claims)
	require.Contains(t, err.Error(), "no license configured for orchestrator")
}

func TestLicenseManager_NewLicenseManager_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	err := os.WriteFile(licensePath, []byte("invalid json content"), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.Error(t, err)
	require.Nil(t, manager)
	require.Contains(t, err.Error(), "failed to parse license file")
}

func TestLicenseManager_NewLicenseManager_FileNotFound(t *testing.T) {
	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: "/non/existent/path/license.json",
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.Error(t, err)
	require.Nil(t, manager)
	require.Contains(t, err.Error(), "failed to read license file")
}

func TestLicenseManager_NewLicenseManager_NilConfig(t *testing.T) {
	manager, err := NewLicenseManager(nil)
	require.Error(t, err)
	require.Nil(t, manager)
	require.Contains(t, err.Error(), "config cannot be nil")
}

func TestLicenseManager_NewLicenseManager_InvalidJSONStructure(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	licenseContent := `{"some_other_field": "value"}`
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	claims, err := manager.ValidateLicense()
	require.Error(t, err)
	require.Nil(t, claims)
	require.Contains(t, err.Error(), "invalid license")
}

func TestLicenseManager_ValidateLicenseWithNodeCount_WithinLimit(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	licenseContent := fmt.Sprintf(`{"license": %q}`, validOfficialTestLicense)
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test with node count equal to limit
	claims, err := manager.ValidateLicenseWithNodeCount(1)
	require.NoError(t, err)
	require.NotNil(t, claims)
	require.Equal(t, "1", claims.Capabilities["max_nodes"])
}

func TestLicenseManager_ValidateLicenseWithNodeCount_ExceedsLimit(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	licenseContent := fmt.Sprintf(`{"license": %q}`, validOfficialTestLicense)
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test with node count exceeding limit
	claims, err := manager.ValidateLicenseWithNodeCount(2)
	require.Error(t, err)
	require.Nil(t, claims)
	require.Contains(t, err.Error(), "node count 2 exceeds licensed limit of 1 nodes")
}

func TestLicenseManager_ValidateLicenseWithNodeCount_NoMaxNodesCapability(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Create a license file with no max_nodes capability
	licenseContent := `{"license": "invalid-token"}`
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	claims, err := manager.ValidateLicenseWithNodeCount(1)
	require.Error(t, err)
	require.Nil(t, claims)
	// The error should be about invalid license first
	require.Contains(t, err.Error(), "invalid license")
}

func TestLicenseManager_ValidateLicenseWithNodeCount_InvalidMaxNodesValue(t *testing.T) {
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "license.json")

	// Create a license with invalid max_nodes value
	licenseContent := `{"license": "invalid-max-nodes-value"}`
	err := os.WriteFile(licensePath, []byte(licenseContent), 0644)
	require.NoError(t, err)

	config := &types.Bacalhau{
		Orchestrator: types.Orchestrator{
			License: types.License{
				LocalPath: licensePath,
			},
		},
	}

	manager, err := NewLicenseManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	claims, err := manager.ValidateLicenseWithNodeCount(1)
	require.Error(t, err)
	require.Nil(t, claims)
	require.Contains(t, err.Error(), "invalid license")
}
