//go:build unit || !integration

package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestAdditiveSet calls sets on the config system sequentially with different values
// and asserts that after each set only the newly set value was added to the config.
// Essentially we are testing two different things:
//   - custom type setting logger.LogMode, types.StorageType, model.JobSelectionDataLocality, etc.
//   - only set values are added to the config.
func TestAdditiveSet(t *testing.T) {
	// this initializes the global viper configuration system
	r := setup.SetupBacalhauRepoForTesting(t)
	repoPath, err := r.Path()
	require.NoError(t, err)
	viper.Set("repo", repoPath)

	current := unmarshalConfigFile(t, repoPath)

	err = setConfig("node.loggingmode", "json")
	require.NoError(t, err)

	expected := current
	expected.Node.LoggingMode = logger.LogModeJSON
	actual := unmarshalConfigFile(t, repoPath)
	require.Equal(t, expected, actual)

	err = setConfig("node.compute.jobselection.probehttp", "example.com")
	require.NoError(t, err)

	expected.Node.Compute.JobSelection.ProbeHTTP = "example.com"
	actual = unmarshalConfigFile(t, repoPath)
	require.Equal(t, expected, actual)

	err = setConfig("node.compute.jobselection.locality", "anywhere")
	require.NoError(t, err)

	expected.Node.Compute.JobSelection.Locality = model.Anywhere
	actual = unmarshalConfigFile(t, repoPath)
	require.Equal(t, expected, actual)

	err = setConfig("node.compute.jobtimeouts.jobnegotiationtimeout", "120s")
	require.NoError(t, err)

	expected.Node.Compute.JobTimeouts.JobNegotiationTimeout = types.Duration(time.Second * 120)
	actual = unmarshalConfigFile(t, repoPath)
	require.Equal(t, expected, actual)

	err = setConfig("node.labels", "foo=bar", "bar=buz", "buz=baz")
	require.NoError(t, err)

	expected.Node.Labels = map[string]string{
		"foo": "bar",
		"bar": "buz",
		"buz": "baz",
	}
	actual = unmarshalConfigFile(t, repoPath)
	require.Equal(t, expected, actual)

	err = setConfig("node.clientapi.host", "1.2.3.4")
	require.NoError(t, err)

	expected.Node.ClientAPI.Host = "1.2.3.4"
	actual = unmarshalConfigFile(t, repoPath)
	require.Equal(t, expected, actual)

}

func TestSetFailure(t *testing.T) {
	// this initializes the global viper configuration system
	r := setup.SetupBacalhauRepoForTesting(t)
	repoPath, err := r.Path()
	require.NoError(t, err)
	viper.Set("repo", repoPath)

	// fails as there are too many values, we expect 1
	err = setConfig("node.loggingmode", "json", "jayson", "mayson", "grayson")
	require.Error(t, err)

	// fails as baeson is not a valid type
	err = setConfig("node.loggingmode", "baeson")
	require.Error(t, err)

	// fails as the key isn't a valid config key
	err = setConfig("not.a.config.key", "porkchop sandwiches")
	require.Error(t, err)
}

func unmarshalConfigFile(t testing.TB, repoPath string) types.BacalhauConfig {
	configFile := filepath.Join(repoPath, config.ConfigFileName)
	v := viper.New()
	v.SetTypeByDefaultValue(true)
	v.SetConfigFile(configFile)

	require.NoError(t, v.ReadInConfig())

	var fileCfg types.BacalhauConfig
	require.NoError(t, v.Unmarshal(&fileCfg, config.DecoderHook))
	return fileCfg
}
