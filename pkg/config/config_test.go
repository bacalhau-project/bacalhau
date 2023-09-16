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

		retrieved, err := Get[string](types.NodeServerAPIHost)
		assert.Nil(t, err)
		assert.Equal(t, expectedConfig.Node.ServerAPI.Host, retrieved)
	})

	// Testing KeyAsEnvVar
	t.Run("KeyAsEnvVar", func(t *testing.T) {
		assert.Equal(t, "BACALHAU_NODE_SERVERAPI_HOST", KeyAsEnvVar(types.NodeServerAPIHost))
	})

	// Testing Init
	t.Run("Init", func(t *testing.T) {
		testCases := []struct {
			name       string
			configType string
		}{
			{"config", "yaml"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				defer Reset()
				expectedConfig := configenv.Testing
				configPath := t.TempDir()

				_, err := Init(expectedConfig, configPath, tc.name, tc.configType)
				require.NoError(t, err)

				var out types.NodeConfig
				err = ForKey(types.Node, &out)
				assert.Nil(t, err)
				assert.Equal(t, expectedConfig.Node.ServerAPI, out.ServerAPI)

				retrieved, err := Get[string](types.NodeServerAPIHost)
				assert.Nil(t, err)
				assert.Equal(t, expectedConfig.Node.ServerAPI.Host, retrieved)
			})
		}

	})

	t.Run("Load", func(t *testing.T) {
		testCases := []struct {
			name       string
			configType string
		}{
			{"yaml config type", "yaml"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				defer Reset()
				// First, set up an expected configuration and save it using Init.
				expectedConfig := configenv.Testing
				configPath := t.TempDir()
				configFile := "config"

				_, err := Init(expectedConfig, configPath, configFile, tc.configType)
				require.NoError(t, err)

				// Now, try to load the configuration we just saved.
				loadedConfig, err := Load(configPath, configFile, tc.configType)
				require.NoError(t, err)

				// After loading, compare the loaded configuration with the expected configuration.
				assert.Equal(t, expectedConfig.Node.ServerAPI, loadedConfig.Node.ServerAPI)

				// Further, test specific parts:
				var out types.APIConfig
				err = ForKey(types.NodeServerAPI, &out)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig.Node.ServerAPI, out)

				retrieved, err := Get[string](types.NodeServerAPIHost)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig.Node.ServerAPI.Host, retrieved)
			})
		}
	})
}
