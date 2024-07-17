//go:build unit || !integration

package cliflags

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestConfig represents our application's configuration structure

type TestConfig struct {
	WebUI struct {
		Enabled bool
		Port    int
	}
	Node struct {
		ID        string
		IPAddress string
	}
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
func loadConfig() (*TestConfig, error) {
	var config TestConfig
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
webui:
  enabled: true
  port: 8080
node:
  id: "node1"
  ipaddress: "192.168.1.1"
`
		configFile, err := createTempConfig(content)
		assert.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", configFile})
		err = cmd.Execute()
		assert.NoError(t, err)

		config, err := loadConfig()
		assert.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
		assert.Equal(t, 8080, config.WebUI.Port)
		assert.Equal(t, "node1", config.Node.ID)
		assert.Equal(t, "192.168.1.1", config.Node.IPAddress)
	})

	// Test case 2: Multiple config files with override
	t.Run("MultipleConfigFiles", func(t *testing.T) {
		viper.Reset()
		baseContent := `
webui:
  enabled: false
  port: 8080
node:
  id: "node1"
`
		overrideContent := `
webui:
  enabled: true
node:
  ipaddress: "192.168.1.2"
`
		baseConfig, err := createTempConfig(baseContent)
		assert.NoError(t, err)
		defer os.Remove(baseConfig)

		overrideConfig, err := createTempConfig(overrideContent)
		assert.NoError(t, err)
		defer os.Remove(overrideConfig)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", baseConfig, "-c", overrideConfig})
		err = cmd.Execute()
		assert.NoError(t, err)

		config, err := loadConfig()
		assert.NoError(t, err)
		// This field was overridden by the second config, so we expect it to be true as defined there.
		assert.True(t, config.WebUI.Enabled)
		assert.Equal(t, 8080, config.WebUI.Port)
		assert.Equal(t, "node1", config.Node.ID)
		assert.Equal(t, "192.168.1.2", config.Node.IPAddress)
	})

	// Test case 3: Config file with dot notation override
	t.Run("ConfigFileWithDotNotation", func(t *testing.T) {
		viper.Reset()
		content := `
webui:
  enabled: false
  port: 8080
node:
  id: "node1"
  ipaddress: "192.168.1.1"
`
		configFile, err := createTempConfig(content)
		assert.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", configFile, "-c", "WebUI.Enabled=true", "-c", "Node.IPAddress=192.168.1.3"})
		err = cmd.Execute()
		assert.NoError(t, err)

		config, err := loadConfig()
		assert.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
		assert.Equal(t, 8080, config.WebUI.Port)
		assert.Equal(t, "node1", config.Node.ID)
		assert.Equal(t, "192.168.1.3", config.Node.IPAddress)
	})

	// Test case 4: Dot notation without value (boolean true)
	t.Run("DotNotationWithoutValue", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", "WebUI.Enabled"})
		err := cmd.Execute()
		assert.NoError(t, err)

		config, err := loadConfig()
		assert.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
	})

	// Test case 5: Multiple dot notation values
	t.Run("MultipleDotNotationValues", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand()
		cmd.SetArgs([]string{
			"-c", "WebUI.Enabled=true",
			"-c", "WebUI.Port=9090",
			"-c", "Node.ID=node2",
			"-c", "Node.IPAddress=192.168.1.5",
		})
		err := cmd.Execute()
		assert.NoError(t, err)

		config, err := loadConfig()
		assert.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
		assert.Equal(t, 9090, config.WebUI.Port)
		assert.Equal(t, "node2", config.Node.ID)
		assert.Equal(t, "192.168.1.5", config.Node.IPAddress)
	})

	// Test case 6: Mixing config file and multiple dot notation values
	// dot notation takes precedence
	t.Run("MixedConfigFileAndDotNotation", func(t *testing.T) {
		viper.Reset()
		content := `
webui:
  enabled: false
  port: 8080
node:
  id: "node1"
  ipaddress: "192.168.1.1"
`
		configFile, err := createTempConfig(content)
		assert.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{
			"-c", configFile,
			"-c", "WebUI.Enabled=true",
			"-c", "Node.IPAddress=192.168.1.5",
			"-c", "WebUI.Port=9090",
		})
		err = cmd.Execute()
		assert.NoError(t, err)

		config, err := loadConfig()
		assert.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
		assert.Equal(t, 9090, config.WebUI.Port)
		assert.Equal(t, "node1", config.Node.ID)
		assert.Equal(t, "192.168.1.5", config.Node.IPAddress)
	})
}
