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
	t.Run("SetAndGetHappyPath", func(t *testing.T) {
		expectedConfig := configenv.Testing
		err := Set(expectedConfig)
		assert.NoError(t, err)

		var out types.NodeConfig
		err = ForKey(types.Node, &out)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig.Node, out)

		retrieved, err := Get[string](types.NodeServerAPIHost)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig.Node.ServerAPI.Host, retrieved)
	})
	t.Run("SetAndGetAdvance", func(t *testing.T) {
		expectedConfig := configenv.Testing
		expectedConfig.Node.IPFS.SwarmAddresses = []string{"1", "2", "3", "4", "5"}
		err := Set(expectedConfig)
		assert.NoError(t, err)

		var out types.IpfsConfig
		err = ForKey(types.NodeIPFS, &out)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig.Node.IPFS, out)

		var node types.NodeConfig
		err = ForKey(types.Node, &node)
		assert.Equal(t, expectedConfig.Node, node)
		assert.NoError(t, err)

		var invalidNode types.NodeConfig
		err = ForKey("INVALID", &invalidNode)
		assert.Error(t, err)
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
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig.Node.ServerAPI, out.ServerAPI)

				retrieved, err := Get[string](types.NodeServerAPIHost)
				assert.NoError(t, err)
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
