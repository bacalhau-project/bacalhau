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

	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

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
func loadConfig() (*types2.Bacalhau, error) {
	configFiles := viper.GetStringSlice(RootCommandConfigFiles)
	for _, f := range configFiles {
		viper.SetConfigFile(f)
		if err := viper.MergeInConfig(); err != nil {
			return nil, err
		}
	}
	base := viper.AllSettings()
	override := viper.GetStringMap(RootCommandConfigValues)
	for k, v := range override {
		base[k] = v
	}
	if err := viper.MergeConfigMap(base); err != nil {
		return nil, err
	}
	var config types2.Bacalhau
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
DataDir: the_name
NameProvider: the_provider
API:
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

		assert.Equal(t, "the_name", config.DataDir)
		assert.Equal(t, "the_provider", config.NameProvider)
		assert.Equal(t, "1.1.1.1", config.API.Host)
		assert.Equal(t, 1111, config.API.Port)
	})

	// Test case 2: Multiple config files with override
	t.Run("MultipleConfigFiles", func(t *testing.T) {
		viper.Reset()
		baseContent := `
  DataDir: the_name
  NameProvider: the_provider
  API:
    Host: 1.1.1.1
    Port: 1111
`
		overrideContent := `

  DataDir: override
  API:
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

		assert.Equal(t, 1111, config.API.Port)
		assert.Equal(t, "the_provider", config.NameProvider)
		// This field was overridden by the second config
		assert.Equal(t, "override", config.DataDir)
		assert.Equal(t, "2.2.2.2", config.API.Host)
	})

	// Test case 3: Config file with dot notation override
	t.Run("ConfigFileWithDotNotation", func(t *testing.T) {
		viper.Reset()
		content := `
  DataDir: the_name
  NameProvider: the_provider
  API:
    Host: 1.1.1.1
    Port: 1111
`
		configFile, err := createTempConfig(content)
		require.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", configFile, "-c", "DataDir=override", "-c", "API.Host=2.2.2.2"})
		err = cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.Equal(t, "the_provider", config.NameProvider)
		assert.Equal(t, 1111, config.API.Port)
		// overrides by flag
		assert.Equal(t, "override", config.DataDir)
		assert.Equal(t, "2.2.2.2", config.API.Host)
	})

	// Test case 4: Dot notation without value (boolean true)
	t.Run("DotNotationWithoutValue", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand()
		cmd.SetArgs([]string{"-c", "WebUI.Enabled"})
		err := cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
	})

	// Test case 5: Multiple dot notation values
	t.Run("MultipleDotNotationValues", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand()
		cmd.SetArgs([]string{
			"-c", "WebUI.Enabled=true",
			"-c", "WebUI.Listen=0.0.0.0:9090",
			"-c", "DataDir=node2",
			"-c", "NameProvider=the_provider",
		})
		err := cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.True(t, config.WebUI.Enabled)
		assert.Equal(t, "0.0.0.0:9090", config.WebUI.Listen)
		assert.Equal(t, "node2", config.DataDir)
		assert.Equal(t, "the_provider", config.NameProvider)
	})

	// Test case 6: Mixing config file and multiple dot notation values
	// dot notation takes precedence
	t.Run("MixedConfigFileAndDotNotation", func(t *testing.T) {
		viper.Reset()
		content := `
  DataDir: the_name
  NameProvider: the_provider
  API:
    Host: 1.1.1.1
    Port: 1111
  WebUI:
    Enabled: false
    Listen: 0.0.0.0:8888
`
		configFile, err := createTempConfig(content)
		require.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand()
		cmd.SetArgs([]string{
			"-c", configFile,
			"-c", "WebUI.Enabled=true",
			"-c", "WebUI.Listen=0.0.0.0:9090",
			"-c", "API.Host=192.168.1.5",
		})
		err = cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)

		assert.Equal(t, "the_name", config.DataDir)
		assert.Equal(t, "the_provider", config.NameProvider)
		assert.Equal(t, 1111, config.API.Port)

		// overrider from flag
		assert.True(t, config.WebUI.Enabled)
		assert.Equal(t, "0.0.0.0:9090", config.WebUI.Listen)
		assert.Equal(t, "192.168.1.5", config.API.Host)
	})

}
