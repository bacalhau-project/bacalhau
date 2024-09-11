//go:build unit || !integration

package config_legacy_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
)

func TestConfig(t *testing.T) {
	// Cleanup viper settings after each test

	// Testing Set and Get
	t.Run("NewWriteRead", func(t *testing.T) {
		// create a testing config instance
		expectedConfig := types.Testing
		wc, err := config_legacy.New(config_legacy.WithDefault(expectedConfig))
		require.NoError(t, err)

		// write the config file to disk
		cfgFilePath := filepath.Join(t.TempDir(), fmt.Sprintf("%d_config.yaml", time.Now().UnixNano()))
		err = wc.Write(cfgFilePath)
		require.NoError(t, err)

		// read the file we wrote from disk
		rc, err := config_legacy.New(config_legacy.WithDefault(expectedConfig))
		require.NoError(t, err)
		err = rc.Load(cfgFilePath)
		require.NoError(t, err)

		wCfg, err := wc.Current()
		require.NoError(t, err)
		rCfg, err := rc.Current()
		require.NoError(t, err)

		assert.Equal(t, expectedConfig.Node.Network, wCfg.Node.Network)
		assert.Equal(t, expectedConfig.Node.Network, rCfg.Node.Network)

	})

	t.Run("SetAndWrite", func(t *testing.T) {
		// create a testing config instance
		expectedConfig := types.Testing
		expectedConfig.Node.Name = "unexpected_name"
		wc, err := config_legacy.New(config_legacy.WithDefault(expectedConfig))
		require.NoError(t, err)

		const expectedName = "bacalhau_testing"
		wc.Set(types.NodeName, expectedName)
		// write the config file to disk
		cfgFilePath := filepath.Join(t.TempDir(), fmt.Sprintf("%d_config.yaml", time.Now().UnixNano()))
		err = wc.Write(cfgFilePath)
		require.NoError(t, err)

		// read the file we wrote from disk
		rc, err := config_legacy.New(config_legacy.WithDefault(expectedConfig))
		require.NoError(t, err)
		err = rc.Load(cfgFilePath)
		require.NoError(t, err)

		wCfg, err := wc.Current()
		require.NoError(t, err)
		rCfg, err := rc.Current()
		require.NoError(t, err)

		assert.Equal(t, expectedName, rCfg.Node.Name)
		assert.Equal(t, expectedName, wCfg.Node.Name)

	})

}
