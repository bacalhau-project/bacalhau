//go:build unit || !integration

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestConfigWithNoValues(t *testing.T) {
	// ensure viper is fresh when test starts and ends
	viper.Reset()
	defer viper.Reset()

	defaultConfig := configenv.Testing

	// Create the config without any file
	cfg, err := New(
		WithDefault(defaultConfig),
	)
	require.NoError(t, err)

	// derive the current config from the above settings
	current, err := cfg.Current()
	require.NoError(t, err)

	// values from the default config are not overridden there wasn't a flag or a file for these
	assert.Equal(t, defaultConfig.Node.ServerAPI.Host, current.Node.ServerAPI.Host, "default value from testing config is expected")
	assert.Equal(t, defaultConfig.Node.ServerAPI.Port, current.Node.ServerAPI.Port, "default value from testing config is expected")
	assert.Equal(t, defaultConfig.Node.ClientAPI.Host, current.Node.ClientAPI.Host, "value provided from client should not be overridden")
	assert.Equal(t, defaultConfig.Node.ClientAPI.Port, current.Node.ClientAPI.Port, "value provided from client should not be overridden")
}
func TestConfigWithNoFile(t *testing.T) {
	// ensure viper is fresh when test starts and ends
	viper.Reset()
	defer viper.Reset()

	defaultConfig := configenv.Testing

	// simulate a user setting a value from the command line with goes to the global viper instance
	// e.g. bacalhau -c node.clientapi.port=123456 -c node.clientapi.host=HOST
	viper.Set(types.NodeClientAPIPort, 123456)
	viper.Set(types.NodeClientAPIHost, "HOST")

	// set and env var to ensure it overrides the default config
	t.Setenv("BACALHAU_NODE_WEBUI_ENABLED", "true")

	// Create the config without any file
	cfg, err := New(
		WithDefault(defaultConfig),
		WithValues(viper.AllSettings()),
	)
	require.NoError(t, err)

	// derive the current config from the above settings
	current, err := cfg.Current()
	require.NoError(t, err)

	// values from the default config are not overridden there wasn't a flag or a file for these
	assert.Equal(t, defaultConfig.Node.ServerAPI.Host, current.Node.ServerAPI.Host, "default value from testing config is expected")
	assert.Equal(t, defaultConfig.Node.ServerAPI.Port, current.Node.ServerAPI.Port, "default value from testing config is expected")

	// values we simulated the client setting are set.
	assert.Equal(t, "HOST", current.Node.ClientAPI.Host, "value provided from client should not be overridden")
	assert.Equal(t, 123456, current.Node.ClientAPI.Port, "value provided from client should not be overridden")

	// values from env vars override default config
	assert.True(t, current.Node.WebUI.Enabled)
}

func TestConfigWithSingleFile(t *testing.T) {
	// ensure viper is fresh when test starts and ends
	viper.Reset()
	defer viper.Reset()

	// Set up temporary directory for config file
	tempDir := t.TempDir()

	defaultConfig := configenv.Testing

	// simulate a user setting a value from the command line with goes to the global viper instance
	// e.g. bacalhau -c node.clientapi.port=123456 -c node.clientapi.host=HOST
	viper.Set(types.NodeClientAPIPort, 123456)
	viper.Set(types.NodeClientAPIHost, "HOST")

	// set and env var to ensure it overrides the default config
	t.Setenv("BACALHAU_NODE_WEBUI_ENABLED", "true")

	// create a single config file
	singleFilePath := filepath.Join(tempDir, FileName)
	writeConfig(t, singleFilePath, Config{
		Node: Node{
			Name:         "single_value",
			NameProvider: "single_value",
			Labels: map[string]string{
				"single": "value",
			},
			ClientAPI: ClientAPI{
				Port: 2222, // overridden by -c
			},
		},
	})

	// Create the config
	cfg, err := New(
		WithDefault(defaultConfig),
		WithValues(viper.AllSettings()),
		WithPaths(singleFilePath),
	)
	require.NoError(t, err)

	// derive the current config from the above settings
	current, err := cfg.Current()
	require.NoError(t, err)

	// values from the default config are not overridden there wasn't a flag or a file for these
	assert.Equal(t, defaultConfig.Node.ServerAPI.Host, current.Node.ServerAPI.Host, "default value from testing config is expected")
	assert.Equal(t, defaultConfig.Node.ServerAPI.Port, current.Node.ServerAPI.Port, "default value from testing config is expected")

	// values we simulated the client setting are set.
	assert.Equal(t, "HOST", current.Node.ClientAPI.Host, "value provided from client should not be overridden")
	assert.Equal(t, 123456, current.Node.ClientAPI.Port, "value provided from client should not be overridden")

	// value from the single config file are persisted
	assert.Equal(t, "single_value", current.Node.Name, "Value from single config should be used")
	assert.Equal(t, "single_value", current.Node.NameProvider, "Value from single config should be used")
	assert.EqualValues(t, map[string]string{"single": "value"}, current.Node.Labels, "values from single config should be used")

	// values from env vars override default config
	assert.True(t, current.Node.WebUI.Enabled)
}

func TestConfigMultipleFiles(t *testing.T) {
	// ensure viper is fresh when test starts and ends
	viper.Reset()
	defer viper.Reset()

	// Set up temporary directory for config files
	tempDir := t.TempDir()

	defaultConfig := configenv.Testing

	// simulate a user setting a value from the command line with goes to the global viper instance
	// e.g. bacalhau -c node.clientapi.port=123456 -c node.clientapi.host=HOST
	viper.Set(types.NodeClientAPIPort, 123456)
	viper.Set(types.NodeClientAPIHost, "HOST")

	// set and env var to ensure it overrides the default config
	t.Setenv("BACALHAU_NODE_WEBUI_ENABLED", "true")

	// create a base config file, we will override parts of this
	baseDir := filepath.Join(tempDir, "base")
	require.NoError(t, os.Mkdir(baseDir, 0755))
	baseFilePath := filepath.Join(baseDir, FileName)
	writeConfig(t, baseFilePath, Config{
		Node: Node{
			Name:         "base_value", // expected to persist
			NameProvider: "base_value", // overridden by override config
			Labels: map[string]string{
				"base": "value", // merged with override config
			},
			ClientAPI: ClientAPI{
				Port: 1111, // overridden by -c
			},
		},
	})

	// create an override config file
	overrideDir := filepath.Join(tempDir, "override")
	require.NoError(t, os.MkdirAll(overrideDir, 0755))
	overrideFilePath := filepath.Join(overrideDir, FileName)
	writeConfig(t, overrideFilePath, Config{
		Node: Node{
			NameProvider: "override_value", // expect to persist
			Labels: map[string]string{
				"override": "value", // merged with base config
			},
			ClientAPI: ClientAPI{
				Port: 2222, // overridden by -c
			},
		},
	})

	// Create the config
	cfg, err := New(
		WithDefault(defaultConfig),
		WithValues(viper.AllSettings()),
		WithPaths(baseFilePath, overrideFilePath),
	)
	require.NoError(t, err)

	// derive the current config from the above settings
	current, err := cfg.Current()
	require.NoError(t, err)

	// values from the default config are not overridden there wasn't a flag or a file for these
	assert.Equal(t, defaultConfig.Node.ServerAPI.Host, current.Node.ServerAPI.Host, "default value from testing config is expected")
	assert.Equal(t, defaultConfig.Node.ServerAPI.Port, current.Node.ServerAPI.Port, "default value from testing config is expected")

	// values we simulated the client setting are set.
	assert.Equal(t, "HOST", current.Node.ClientAPI.Host, "value provided from client should not be overridden")
	assert.Equal(t, 123456, current.Node.ClientAPI.Port, "value provided from client should not be overridden")

	// value from the base config are persisted
	assert.Equal(t, "base_value", current.Node.Name, "Value from base config should be used")

	// override value overrides the base config.
	assert.Equal(t, "override_value", current.Node.NameProvider, "Value from override should be used")

	// values from the base config and override config are merged together.
	assert.EqualValues(t, map[string]string{"override": "value", "base": "value"}, current.Node.Labels, "values from base and override should be merged")

	// values from env vars override default config
	assert.True(t, current.Node.WebUI.Enabled)

}

// a subset of the whole bacalhau config
type Config struct {
	Node Node `yaml:"Node"`
}

type Node struct {
	Name         string            `yaml:"Name,omitempty"`
	NameProvider string            `yaml:"NameProvider,omitempty"`
	Labels       map[string]string `yaml:"Labels,omitempty"`
	ClientAPI    `yaml:"ClientAPI,omitempty"`
}

type ClientAPI struct {
	Host string `yaml:"Host,omitempty"`
	Port int    `yaml:"Port,omitempty"`
}

func writeConfig(t *testing.T, filename string, config Config) {
	data, err := yaml.Marshal(&config)
	require.NoError(t, err)

	err = os.WriteFile(filename, data, 0644)
	require.NoError(t, err)
}
