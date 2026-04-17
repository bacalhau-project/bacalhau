package node

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
)

func TestNodeConfig_Validate_NodeID(t *testing.T) {
	t.Run("Valid NodeID should not return error", func(t *testing.T) {
		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         types.Bacalhau{},
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Validate() should not return an error for valid NodeID")
	})

	t.Run("Empty NodeID should return error", func(t *testing.T) {
		cfg := &NodeConfig{
			NodeID:                 "",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         types.Bacalhau{},
		}

		err := cfg.Validate()
		assert.Error(t, err, "Validate() should return an error for empty NodeID")
		assert.Contains(t, err.Error(), "node id is required", "Error message should mention the nodeID requirement")
	})
}

func TestNodeConfig_Validate_AuthConfig_ValidCases(t *testing.T) {
	t.Run("Empty AuthConfig should be valid", func(t *testing.T) {
		authConfig := types.AuthConfig{}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Validate() should not return an error for empty AuthConfig")
	})

	t.Run("AuthConfig with only Methods should be valid", func(t *testing.T) {
		authConfig := types.AuthConfig{
			Methods: map[string]types.AuthenticatorConfig{
				"test": {
					Type:       "test-type",
					PolicyPath: "test-path",
				},
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Validate() should not return an error for AuthConfig with only Methods")
	})

	t.Run("AuthConfig with only AccessPolicyPath should be valid", func(t *testing.T) {
		authConfig := types.AuthConfig{
			AccessPolicyPath: "/path/to/policy",
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Validate() should not return an error for AuthConfig with only AccessPolicyPath")
	})

	t.Run("AuthConfig with only Users should be valid", func(t *testing.T) {
		authConfig := types.AuthConfig{
			Users: []types.AuthUser{
				{
					Username: "test-user",
					Password: "test-password",
				},
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Validate() should not return an error for AuthConfig with only Users")
	})

	t.Run("AuthConfig with only Oauth2 should be valid", func(t *testing.T) {
		authConfig := types.AuthConfig{
			Oauth2: types.Oauth2Config{
				ProviderID: "test-provider",
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.NoError(t, err, "Validate() should not return an error for AuthConfig with only Oauth2")
	})
}

func TestNodeConfig_Validate_AuthConfig_InvalidCases(t *testing.T) {
	t.Run("AuthConfig with Users and AccessPolicyPath should be invalid", func(t *testing.T) {
		authConfig := types.AuthConfig{
			AccessPolicyPath: "/path/to/policy",
			Users: []types.AuthUser{
				{
					Username: "test-user",
					Password: "test-password",
				},
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.Error(t, err, "Validate() should return an error for AuthConfig with Users and AccessPolicyPath")
		assert.Contains(t, err.Error(), "mixing old and new auth mechanisms", "Error message should describe the issue")
	})

	t.Run("AuthConfig with Oauth2 and AccessPolicyPath should be invalid", func(t *testing.T) {
		authConfig := types.AuthConfig{
			AccessPolicyPath: "/path/to/policy",
			Oauth2: types.Oauth2Config{
				ProviderID: "test-provider",
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.Error(t, err, "Validate() should return an error for AuthConfig with Oauth2 and AccessPolicyPath")
		assert.Contains(t, err.Error(), "mixing old and new auth mechanisms", "Error message should describe the issue")
	})

	t.Run("AuthConfig with Users, Oauth2, Methods, and AccessPolicyPath should be invalid", func(t *testing.T) {
		authConfig := types.AuthConfig{
			Methods: map[string]types.AuthenticatorConfig{
				"test": {
					Type:       "test-type",
					PolicyPath: "test-path",
				},
			},
			AccessPolicyPath: "/path/to/policy",
			Users: []types.AuthUser{
				{
					Username: "test-user",
					Password: "test-password",
				},
			},
			Oauth2: types.Oauth2Config{
				ProviderID: "test-provider",
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.Error(t, err, "Validate() should return an error for AuthConfig with all fields populated")
		assert.Contains(t, err.Error(), "mixing old and new auth mechanisms", "Error message should describe the issue")
	})

	t.Run("AuthConfig with different Oauth2 field (not ProviderID) should be detected", func(t *testing.T) {
		authConfig := types.AuthConfig{
			AccessPolicyPath: "/path/to/policy",
			Oauth2: types.Oauth2Config{
				// Using a different field than ProviderID
				ProviderName: "test-provider-name",
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.Error(t, err, "Validate() should detect any non-zero Oauth2 field")
		assert.Contains(t, err.Error(), "mixing old and new auth mechanisms", "Error message should describe the issue")
	})

	t.Run("AuthConfig with Oauth2.Scopes (slice field) populated should be detected", func(t *testing.T) {
		authConfig := types.AuthConfig{
			AccessPolicyPath: "/path/to/policy",
			Oauth2: types.Oauth2Config{
				Scopes: []string{"scope1", "scope2"},
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.Error(t, err, "Validate() should detect populated slice in Oauth2")
		assert.Contains(t, err.Error(), "mixing old and new auth mechanisms", "Error message should describe the issue")
	})

	t.Run("AuthConfig with Oauth2.PollingInterval (int field) populated should be detected", func(t *testing.T) {
		authConfig := types.AuthConfig{
			AccessPolicyPath: "/path/to/policy",
			Oauth2: types.Oauth2Config{
				PollingInterval: 30,
			},
		}
		bacConfig := types.Bacalhau{}
		bacConfig.API.Auth = authConfig

		cfg := &NodeConfig{
			NodeID:                 "test-node-id",
			CleanupManager:         system.NewCleanupManager(),
			FailureInjectionConfig: models.FailureInjectionConfig{},
			BacalhauConfig:         bacConfig,
		}

		err := cfg.Validate()
		assert.Error(t, err, "Validate() should detect populated integer in Oauth2")
		assert.Contains(t, err.Error(), "mixing old and new auth mechanisms", "Error message should describe the issue")
	})
}
