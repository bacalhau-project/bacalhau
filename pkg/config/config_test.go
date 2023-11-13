//go:build unit || !integration

package config

import (
	"os"
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

				// since BACALHAU_ENVIRONMENT is set to testing we exptect the testing config
				require.NoError(t, os.Setenv("BACALHAU_ENVIRONMENT", "test"))
				expected := configenv.Testing

				configPath := t.TempDir()

				_, err := Init(configPath)
				require.NoError(t, err)

				var out types.NodeConfig
				err = ForKey(types.Node, &out)
				assert.NoError(t, err)
				assert.Equal(t, expected.Node.Requester, out.Requester)

				var retrieved types.RequesterConfig
				require.NoError(t, ForKey(types.NodeRequester, &retrieved))
				assert.Equal(t, expected.Node.Requester, retrieved)
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
				require.NoError(t, os.Setenv("BACALHAU_ENVIRONMENT", "test"))
				// since BACALHAU_ENVIRONMENT is set to testing we exptect the testing config
				expected := configenv.Testing
				configPath := t.TempDir()

				_, err := Init(configPath)
				require.NoError(t, err)

				// Now, try to load the configuration we just saved.
				loadedConfig, err := Load(configPath)
				require.NoError(t, err)

				// After loading, compare the loaded configuration with the expected configuration.
				assert.Equal(t, expected.Node.Requester, loadedConfig.Node.Requester)

				// Further, test specific parts:
				var out types.NodeConfig
				err = ForKey(types.Node, &out)
				assert.NoError(t, err)
				assert.Equal(t, expected.Node.Requester, out.Requester)

				var retrieved types.RequesterConfig
				require.NoError(t, ForKey(types.NodeRequester, &retrieved))
				assert.Equal(t, expected.Node.Requester, retrieved)
			})
		}
	})
}
