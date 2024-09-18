//go:build unit || !integration

package config_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	cmd2 "github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestSetWithExplicitConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	explicitConfigPath := filepath.Join(tempDir, "explicit_config.yaml")

	// Create an empty config file
	_, err := os.Create(explicitConfigPath)
	require.NoError(t, err)

	cmd := cmd2.NewRootCmd()
	cmd.SetArgs([]string{"config", "set", "--config", explicitConfigPath, "api.host", "2.2.2.2"})

	err = cmd.Execute()
	require.NoError(t, err)

	actual := unmarshalConfigFile(t, explicitConfigPath)
	expected := types.Bacalhau{API: types.API{Host: "2.2.2.2"}}
	require.Equal(t, expected, actual)
}

func TestSetWithDefaultConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	cmd := cmd2.NewRootCmd()
	cmd.SetArgs([]string{"config", "set", "api.port", "5678"})

	err := cmd.Execute()
	require.NoError(t, err)

	defaultConfigPath := filepath.Join(tempDir, "config.yaml")
	actual := unmarshalConfigFile(t, defaultConfigPath)
	expected := types.Bacalhau{API: types.API{Port: 5678}}
	require.Equal(t, expected, actual)
}

func TestSetMultipleValues(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("BACALHAU_DIR", tempDir)
	defer os.Unsetenv("BACALHAU_DIR")

	cmd := cmd2.NewRootCmd()
	cmd.SetArgs([]string{"config", "set", "compute.orchestrators", "http://127.0.0.1:1234", "http://1.1.1.1:1234"})

	err := cmd.Execute()
	require.NoError(t, err)

	defaultConfigPath := filepath.Join(tempDir, "config.yaml")
	actual := unmarshalConfigFile(t, defaultConfigPath)
	expected := types.Bacalhau{Compute: types.Compute{
		Orchestrators: []string{
			"http://127.0.0.1:1234",
			"http://1.1.1.1:1234",
		},
	}}
	require.Equal(t, expected, actual)
}

func TestSetInvalidKey(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BACALHAU_DIR", tempDir)

	cmd := cmd2.NewRootCmd()
	cmd.SetArgs([]string{"config", "set", "invalid.key", "value"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a valid config key")
}

func TestSetAdditiveChanges(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create an empty config file
	_, err := os.Create(configPath)
	require.NoError(t, err)

	// First set
	cmd1 := cmd2.NewRootCmd()
	cmd1.SetArgs([]string{"config", "set", "--config", configPath, "api.host", "127.0.0.1"})
	err = cmd1.Execute()
	require.NoError(t, err)

	// Second set
	cmd2 := cmd2.NewRootCmd()
	cmd2.SetArgs([]string{"config", "set", "--config", configPath, "api.port", "1234"})
	err = cmd2.Execute()
	require.NoError(t, err)

	actual := unmarshalConfigFile(t, configPath)
	expected := types.Bacalhau{API: types.API{
		Host: "127.0.0.1",
		Port: 1234,
	}}
	require.Equal(t, expected, actual)
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
