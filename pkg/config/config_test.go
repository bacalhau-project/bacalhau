//go:build unit || !integration

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestConfig(t *testing.T) {
	// Cleanup viper settings after each test
	defer Reset()

	// Testing Set and Get
	t.Run("SetAndGet", func(t *testing.T) {
		expectedConfig := configenv.Testing
		err := Set(expectedConfig)
		assert.Nil(t, err)

		var out types.NodeConfig
		err = ForKey(types.Node, &out)
		assert.Nil(t, err)
		assert.Equal(t, expectedConfig.Node, out)

		retrieved, err := Get[string](types.NodeAPIHost)
		assert.Nil(t, err)
		assert.Equal(t, expectedConfig.Node.API.Host, retrieved)
	})

	// Testing KeyAsEnvVar
	t.Run("KeyAsEnvVar", func(t *testing.T) {
		assert.Equal(t, "BACALHAU_NODE_API_HOST", KeyAsEnvVar(types.NodeAPIHost))
	})

	// Testing Init
	t.Run("Init", func(t *testing.T) {
		testCases := []struct {
			name       string
			configType string
		}{
			{"config", "yaml"},
			{"config", "toml"},
			{"config", "json"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				defer Reset()
				expectedConfig := configenv.Testing
				configPath := t.TempDir()

				_, err := Init(&expectedConfig, configPath, tc.name, tc.configType)
				require.NoError(t, err)

				var out types.NodeConfig
				err = ForKey(types.Node, &out)
				assert.Nil(t, err)
				assert.Equal(t, expectedConfig.Node, out)

				retrieved, err := Get[string](types.NodeAPIHost)
				assert.Nil(t, err)
				assert.Equal(t, expectedConfig.Node.API.Host, retrieved)
			})
		}

	})

	t.Run("Load", func(t *testing.T) {
		testCases := []struct {
			name       string
			configType string
		}{
			{"yaml config type", "yaml"},
			{"toml config type", "toml"},
			{"json config type", "json"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				defer Reset()
				// First, set up an expected configuration and save it using Init.
				expectedConfig := configenv.Testing
				configPath := t.TempDir()
				configFile := "config"

				_, err := Init(&expectedConfig, configPath, configFile, tc.configType)
				require.NoError(t, err)

				// Now, try to load the configuration we just saved.
				loadedConfig, err := Load(configPath, configFile, tc.configType)
				require.NoError(t, err)

				// After loading, compare the loaded configuration with the expected configuration.
				assert.Equal(t, expectedConfig.Node.API, loadedConfig.Node.API)

				// Further, test specific parts:
				var out types.APIConfig
				err = ForKey(types.NodeAPI, &out)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig.Node.API, out)

				retrieved, err := Get[string](types.NodeAPIHost)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig.Node.API.Host, retrieved)
			})
		}
	})
}
