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

func TestConfigPrecedence(t *testing.T) {
	// ensure viper is fresh when test starts and ends
	viper.Reset()
	defer viper.Reset()

	// Set up temporary directories for repo and XDG configs
	tempDir := t.TempDir()

	defaultConfig := configenv.Testing

	// simulate a user setting a value from the command line with goes to the global viper instance
	// e.g. bacalhau -c node.clientapi.port=123456 -c node.clientapi.host=HOST
	viper.Set(types.NodeClientAPIPort, 123456)
	viper.Set(types.NodeClientAPIHost, "HOST")

	// set and env var to ensure it overrides the default config
	t.Setenv("BACALHAU_NODE_WEBUI_ENABLED", "true")

	// create a bacalhau repo with a config file
	repoDir := filepath.Join(tempDir, "repo")
	require.NoError(t, os.Mkdir(repoDir, 0755))
	viper.Set("repo", repoDir)
	repoConfigPath := filepath.Join(repoDir, FileName)
	writeConfig(t, repoConfigPath, Config{
		Node: Node{
			Name:         "repo_value", // expected to persist
			NameProvider: "repo_value", // overridden by xdg config
			Labels: map[string]string{
				"repo": "value", // merged with xdg config
			},
			ClientAPI: ClientAPI{
				Port: 1111, // overridden by -c
			},
		},
	})

	// create a config file in xdg config dir
	xdgDir := filepath.Join(tempDir, "xdg", "bacalhau")
	require.NoError(t, os.MkdirAll(xdgDir, 0755))
	xdgConfigPath := filepath.Join(xdgDir, FileName)
	writeConfig(t, xdgConfigPath, Config{
		Node: Node{
			NameProvider: "xdg_value", // expect to persist
			Labels: map[string]string{
				"xdg": "value", // merged with bacalhau repo
			},
			ClientAPI: ClientAPI{
				Port: 2222, // overridden by -c
			},
		},
	})

	// Create the config
	cfg := New(
		WithDefaultConfig(defaultConfig),
		WithRepoPath(repoConfigPath),
		WithXDGPath(xdgConfigPath),
	)

	// derive the current config from the above settings
	current, err := cfg.Current()
	require.NoError(t, err)

	// values from the default config are not overridden there wasn't a flag or a file for these
	assert.Equal(t, defaultConfig.Node.ServerAPI.Host, current.Node.ServerAPI.Host, "default value from testing config is expected")
	assert.Equal(t, defaultConfig.Node.ServerAPI.Port, current.Node.ServerAPI.Port, "default value from testing config is expected")

	// values we simulated the client setting are set.
	assert.Equal(t, "HOST", current.Node.ClientAPI.Host, "value provided from client should not be overridden")
	assert.Equal(t, 123456, current.Node.ClientAPI.Port, "value provided from client should not be overridden")

	// value from the repo config are persisted
	assert.Equal(t, "repo_value", current.Node.Name, "Value from repo config should be used")

	// xdg value overrides the repo config.
	assert.Equal(t, "xdg_value", current.Node.NameProvider, "XDG config should override repo config")

	// values from the repo config and xdg config are merged together.
	assert.EqualValues(t, map[string]string{"xdg": "value", "repo": "value"}, current.Node.Labels, "values from repo and xdg should be merged")

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
