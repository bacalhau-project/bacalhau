//go:build unit || !integration

package config_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestConfig(t *testing.T) {
	// Testing Set and Get
	t.Run("SetAndGetHappyPath", func(t *testing.T) {
		expectedConfig := configenv.Testing
		cfg := config.New(config.WithDefaultConfig(expectedConfig))

		var out types.NodeConfig
		err := cfg.ForKey(types.Node, &out)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig.Node, out)

		var retrieved string
		err = cfg.ForKey(types.NodeServerAPIHost, &retrieved)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig.Node.ServerAPI.Host, retrieved)
	})
	t.Run("SetAndGetAdvance", func(t *testing.T) {
		expectedConfig := configenv.Testing
		expectedConfig.Node.IPFS.SwarmAddresses = []string{"1", "2", "3", "4", "5"}
		cfg := config.New(config.WithDefaultConfig(expectedConfig))

		var out types.IpfsConfig
		err := cfg.ForKey(types.NodeIPFS, &out)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig.Node.IPFS, out)

		var node types.NodeConfig
		err = cfg.ForKey(types.Node, &node)
		assert.Equal(t, expectedConfig.Node, node)
		assert.NoError(t, err)

		var invalidNode types.NodeConfig
		err = cfg.ForKey("INVALID", &invalidNode)
		assert.Error(t, err)
	})

	// Testing KeyAsEnvVar
	t.Run("KeyAsEnvVar", func(t *testing.T) {
		assert.Equal(t, "BACALHAU_NODE_SERVERAPI_HOST", config.KeyAsEnvVar(types.NodeServerAPIHost))
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
				expected := configenv.Testing
				cfg := config.New(config.WithDefaultConfig(expected))

				var out types.NodeConfig
				err := cfg.ForKey(types.Node, &out)
				assert.NoError(t, err)
				assert.Equal(t, expected.Node.Requester, out.Requester)

				var retrieved types.RequesterConfig
				require.NoError(t, cfg.ForKey(types.NodeRequester, &retrieved))
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
				expected := configenv.Testing
				configPath := t.TempDir()
				expected.Node.Requester.JobStore.Path = filepath.Join(configPath, config.OrchestratorJobStorePath)
				cfg := config.New(config.WithDefaultConfig(expected))

				// Now, try to load the configuration we just saved.
				loadedConfig, err := cfg.Current()
				require.NoError(t, err)

				// After loading, compare the loaded configuration with the expected configuration.
				assert.Equal(t, expected.Node.Requester, loadedConfig.Node.Requester)

				// Further, test specific parts:
				var out types.NodeConfig
				err = cfg.ForKey(types.Node, &out)
				assert.NoError(t, err)
				assert.Equal(t, expected.Node.Requester, out.Requester)

				var retrieved types.RequesterConfig
				require.NoError(t, cfg.ForKey(types.NodeRequester, &retrieved))
				assert.Equal(t, expected.Node.Requester, retrieved)
			})
		}
	})
}
