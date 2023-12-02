//go:build unit || !integration

package config

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
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

	err = setConfig("node.loggingmode", "json")
	require.NoError(t, err)

	expected := types.BacalhauConfig{Node: types.NodeConfig{LoggingMode: logger.LogModeJSON}}
	actual := unmarshalConfigFile(t, repoPath)

	require.Equal(t, expected, actual)

	err = setConfig("node.compute.executionstore.type", "boltdb")
	require.NoError(t, err)

	expected = types.BacalhauConfig{Node: types.NodeConfig{
		LoggingMode: logger.LogModeJSON,
		Compute: types.ComputeConfig{
			ExecutionStore: types.JobStoreConfig{
				Type: types.BoltDB,
			},
		},
	}}
	actual = unmarshalConfigFile(t, repoPath)

	require.Equal(t, expected, actual)

	err = setConfig("node.compute.jobselection.locality", "anywhere")
	require.NoError(t, err)

	expected = types.BacalhauConfig{Node: types.NodeConfig{
		LoggingMode: logger.LogModeJSON,
		Compute: types.ComputeConfig{
			ExecutionStore: types.JobStoreConfig{
				Type: types.BoltDB,
			},
			JobSelection: model.JobSelectionPolicy{
				Locality: model.Anywhere,
			},
		},
	}}
	actual = unmarshalConfigFile(t, repoPath)

	require.Equal(t, expected, actual)

	err = setConfig("node.compute.jobtimeouts.jobnegotiationtimeout", "120s")
	require.NoError(t, err)

	expected = types.BacalhauConfig{Node: types.NodeConfig{
		LoggingMode: logger.LogModeJSON,
		Compute: types.ComputeConfig{
			ExecutionStore: types.JobStoreConfig{
				Type: types.BoltDB,
			},
			JobSelection: model.JobSelectionPolicy{
				Locality: model.Anywhere,
			},
			JobTimeouts: types.JobTimeoutConfig{
				JobNegotiationTimeout: types.Duration(time.Second * 120),
			},
		},
	}}
	actual = unmarshalConfigFile(t, repoPath)

	require.Equal(t, expected, actual)

	err = setConfig("node.labels", "foo=bar", "bar=buz", "buz=baz")
	require.NoError(t, err)

	expected = types.BacalhauConfig{Node: types.NodeConfig{
		LoggingMode: logger.LogModeJSON,
		Compute: types.ComputeConfig{
			ExecutionStore: types.JobStoreConfig{
				Type: types.BoltDB,
			},
			JobSelection: model.JobSelectionPolicy{
				Locality: model.Anywhere,
			},
			JobTimeouts: types.JobTimeoutConfig{
				JobNegotiationTimeout: types.Duration(time.Second * 120),
			},
		},
		Labels: map[string]string{
			"foo": "bar",
			"bar": "buz",
			"buz": "baz",
		},
	}}
	actual = unmarshalConfigFile(t, repoPath)

	require.Equal(t, expected, actual)

	err = setConfig("node.clientapi.host", "0.0.0.0")
	require.NoError(t, err)

	expected = types.BacalhauConfig{Node: types.NodeConfig{
		ClientAPI: types.APIConfig{
			Host: "0.0.0.0",
		},
		LoggingMode: logger.LogModeJSON,
		Compute: types.ComputeConfig{
			ExecutionStore: types.JobStoreConfig{
				Type: types.BoltDB,
			},
			JobSelection: model.JobSelectionPolicy{
				Locality: model.Anywhere,
			},
			JobTimeouts: types.JobTimeoutConfig{
				JobNegotiationTimeout: types.Duration(time.Second * 120),
			},
		},
		Labels: map[string]string{
			"foo": "bar",
			"bar": "buz",
			"buz": "baz",
		},
	}}
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

	configPath := filepath.Join(repoPath, "config.yaml")
	configFile, err := os.Open(configPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		configFile.Close()
	})
	configData, err := io.ReadAll(configFile)
	require.NoError(t, err)
	var cfg types.BacalhauConfig
	err = yaml.Unmarshal(configData, &cfg)
	require.NoError(t, err)
	return cfg
}
