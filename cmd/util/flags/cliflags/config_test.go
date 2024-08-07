//go:build unit || !integration

package cliflags

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig represents our application's configuration structure

// a subset of the whole bacalhau config
type Config struct {
	Node Node `yaml:"Node"`
}

type Node struct {
	Name         string    `yaml:"Name,omitempty"`
	NameProvider string    `yaml:"NameProvider,omitempty"`
	ClientAPI    ClientAPI `yaml:"ClientAPI,omitempty"`
	WebUI        WebUI     `yaml:"WebUI,omitempty"`
}

type WebUI struct {
	Enabled bool `yaml:"Enabled,omitempty"`
	Port    int  `yaml:"Port,omitempty"`
}

type ClientAPI struct {
	Host string `yaml:"Host,omitempty"`
	Port int    `yaml:"Port,omitempty"`
}

// createTempConfig creates a temporary config file with the given content
func createTempConfig(content string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "config*.yaml")
	if err != nil {
		return "", err
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

// setupTestCommand sets up a cobra command for testing
func setupTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	cmd.PersistentFlags().VarP(NewConfigFlag(), "config", "c", "config file(s) or dot separated path(s) to config values")
	return cmd
}

// loadConfig loads the configuration from viper
func loadConfig() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func TestConfigMerging(t *testing.T) {
	// Test case 1: Single config file
	t.Run("SingleConfigFile", func(t *testing.T) {
		viper.Reset()
		content := `
Node:
  Name: the_name
  NameProvider: the_provider
  ClientAPI:
    Host: 1.1.1.1
    Port: 1111
`
		configFile, err := createTempConfig(content)
		require.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", configFile})
		err = cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.Equal(t, "the_name", config.Node.Name)
		assert.Equal(t, "the_provider", config.Node.NameProvider)
		assert.Equal(t, "1.1.1.1", config.Node.ClientAPI.Host)
		assert.Equal(t, 1111, config.Node.ClientAPI.Port)
	})

	// Test case 2: Multiple config files with override
	t.Run("MultipleConfigFiles", func(t *testing.T) {
		viper.Reset()
		baseContent := `
Node:
  Name: the_name
  NameProvider: the_provider
  ClientAPI:
    Host: 1.1.1.1
    Port: 1111
`
		overrideContent := `
Node:
  Name: override
  ClientAPI:
    Host: 2.2.2.2
`
		baseConfig, err := createTempConfig(baseContent)
		require.NoError(t, err)
		defer os.Remove(baseConfig)

		overrideConfig, err := createTempConfig(overrideContent)
		require.NoError(t, err)
		defer os.Remove(overrideConfig)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", baseConfig, "-c", overrideConfig})
		err = cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.Equal(t, 1111, config.Node.ClientAPI.Port)
		assert.Equal(t, "the_provider", config.Node.NameProvider)
		// This field was overridden by the second config
		assert.Equal(t, "override", config.Node.Name)
		assert.Equal(t, "2.2.2.2", config.Node.ClientAPI.Host)
	})

	// Test case 3: Config file with dot notation override
	t.Run("ConfigFileWithDotNotation", func(t *testing.T) {
		viper.Reset()
		content := `
Node:
  Name: the_name
  NameProvider: the_provider
  ClientAPI:
    Host: 1.1.1.1
    Port: 1111
`
		configFile, err := createTempConfig(content)
		require.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", configFile, "-c", "Node.Name=override", "-c", "Node.ClientAPI.Host=2.2.2.2"})
		err = cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.Equal(t, "the_provider", config.Node.NameProvider)
		assert.Equal(t, 1111, config.Node.ClientAPI.Port)
		// overrides by flag
		assert.Equal(t, "override", config.Node.Name)
		assert.Equal(t, "2.2.2.2", config.Node.ClientAPI.Host)
	})

	// Test case 4: Dot notation without value (boolean true)
	t.Run("DotNotationWithoutValue", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", "Node.WebUI.Enabled"})
		err := cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)
		assert.True(t, config.Node.WebUI.Enabled)
	})

	// Test case 5: Multiple dot notation values
	t.Run("MultipleDotNotationValues", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand()
		cmd.SetArgs([]string{
			"-c", "Node.WebUI.Enabled=true",
			"-c", "Node.WebUI.Port=9090",
			"-c", "Node.Name=node2",
			"-c", "Node.NameProvider=the_provider",
		})
		err := cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.True(t, config.Node.WebUI.Enabled)
		assert.Equal(t, 9090, config.Node.WebUI.Port)
		assert.Equal(t, "node2", config.Node.Name)
		assert.Equal(t, "the_provider", config.Node.NameProvider)
	})

	// Test case 6: Mixing config file and multiple dot notation values
	// dot notation takes precedence
	t.Run("MixedConfigFileAndDotNotation", func(t *testing.T) {
		viper.Reset()
		content := `
Node:
  Name: the_name
  NameProvider: the_provider
  ClientAPI:
    Host: 1.1.1.1
    Port: 1111
  WebUI:
    Enabled: false
    Port: 8888
`
		configFile, err := createTempConfig(content)
		require.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{
			"-c", configFile,
			"-c", "Node.WebUI.Enabled=true",
			"-c", "Node.WebUI.Port=9090",
			"-c", "Node.ClientAPI.Host=192.168.1.5",
		})
		err = cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.Equal(t, "the_name", config.Node.Name)
		assert.Equal(t, "the_provider", config.Node.NameProvider)
		assert.Equal(t, 1111, config.Node.ClientAPI.Port)

		// overrider from flag
		assert.True(t, config.Node.WebUI.Enabled)
		assert.Equal(t, 9090, config.Node.WebUI.Port)
		assert.Equal(t, "192.168.1.5", config.Node.ClientAPI.Host)
	})

}
