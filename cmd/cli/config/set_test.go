//go:build unit || !integration

package config

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// TestAdditiveSet calls sets on the config system sequentially with different values
// and asserts that after each set only the newly set value was added to the config.
// Essentially we are testing two different things:
func TestAdditiveSet(t *testing.T) {
	cfgFilePath := filepath.Join(t.TempDir(), "config.yaml")

	err := setConfig(cfgFilePath, "api.host", "127.0.0.1")
	require.NoError(t, err)
	err = setConfig(cfgFilePath, "api.port", "1234")
	require.NoError(t, err)

	expected := types.Bacalhau{API: types.API{
		Host: "127.0.0.1",
		Port: 1234,
	}}
	actual := unmarshalConfigFile(t, cfgFilePath)

	require.Equal(t, expected, actual)

	err = setConfig(cfgFilePath, "compute.enabled", "true")
	require.NoError(t, err)
	err = setConfig(cfgFilePath, "compute.orchestrators", "http://127.0.0.1:1234", "http://1.1.1.1:1234")
	require.NoError(t, err)

	expected = types.Bacalhau{
		API: types.API{
			Host: "127.0.0.1",
			Port: 1234,
		},
		Compute: types.Compute{
			Enabled: true,
			Orchestrators: []string{
				"http://127.0.0.1:1234",
				"http://1.1.1.1:1234",
			},
		},
	}
	actual = unmarshalConfigFile(t, cfgFilePath)

	require.Equal(t, expected, actual)

}

func TestSetFailure(t *testing.T) {
	cfgFilePath := filepath.Join(t.TempDir(), "config.yaml")
	// fails as the key isn't a valid config key
	err := setConfig(cfgFilePath, "not.a.config.key", "porkchop sandwiches")
	require.Error(t, err)
}

func unmarshalConfigFile(t testing.TB, path string) types.Bacalhau {

	configFile, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() {
		configFile.Close()
	})
	configData, err := io.ReadAll(configFile)
	require.NoError(t, err)
	var cfg types.Bacalhau
	err = yaml.Unmarshal(configData, &cfg)
	require.NoError(t, err)
	return cfg
}
