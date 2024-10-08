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

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
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

// setupTestCommand to accept a boolean for write mode
func setupTestCommand(writeMode bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {},
	}
	if writeMode {
		cmd.PersistentFlags().VarP(NewWriteConfigFlag(), "config", "c", "config file for writing")
	} else {
		cmd.PersistentFlags().VarP(NewConfigFlag(), "config", "c", "config file(s) or dot separated path(s) to config values")
	}
	return cmd
}

// loadConfig loads the configuration from viper
func loadConfig() (*types.Bacalhau, error) {
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
	var config types.Bacalhau
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

		cmd := setupTestCommand(false)
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

		cmd := setupTestCommand(false)
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

		cmd := setupTestCommand(false)
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
	t.Run("DotNotationBoolWithoutValue", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand(false)
		cmd.SetArgs([]string{"-c", "WebUI.Enabled"})
		err := cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
	})

	// Test case 4.1: Dot notation with value (boolean true)
	t.Run("DotNotationBoolWithValue", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand(false)
		cmd.SetArgs([]string{"-c", "WebUI.Enabled=true"})
		err := cmd.Execute()
		require.NoError(t, err)

		config, err := loadConfig()
		require.NoError(t, err)
		assert.True(t, config.WebUI.Enabled)
	})

	// Test case 4.2: Dot notation without value (non boolean)
	t.Run("DotNotationBoolWithValue", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand(false)
		cmd.SetArgs([]string{"-c", "api.host", "0.0.0.0"})
		err := cmd.Execute()
		require.Error(t, err)
	})

	// Test case 5: Multiple dot notation values
	t.Run("MultipleDotNotationValues", func(t *testing.T) {
		viper.Reset()
		cmd := setupTestCommand(false)
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

		cmd := setupTestCommand(false)
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

	// New test case: Write mode with single config file
	t.Run("WriteModeWithSingleConfigFile", func(t *testing.T) {
		viper.Reset()
		configFile, err := createTempConfig("")
		require.NoError(t, err)
		defer os.Remove(configFile)

		cmd := setupTestCommand(true)
		cmd.SetArgs([]string{"-c", configFile})
		err = cmd.Execute()
		require.NoError(t, err)

		assert.Equal(t, configFile, viper.GetString(RootCommandConfigFiles))
	})

	// New test case: Write mode with multiple config files (should fail)
	t.Run("WriteModeWithMultipleConfigFiles", func(t *testing.T) {
		viper.Reset()
		configFile1, err := createTempConfig("")
		require.NoError(t, err)
		defer os.Remove(configFile1)

		configFile2, err := createTempConfig("")
		require.NoError(t, err)
		defer os.Remove(configFile2)

		cmd := setupTestCommand(true)
		cmd.SetArgs([]string{"-c", configFile1, "-c", configFile2})
		err = cmd.Execute()
		assert.Error(t, err) // Expect an error when trying to use multiple config files in write mode
	})

	// New test case: Write mode with key/value pairs or non-files (should fail)
	t.Run("WriteModeWithKeyValueOrNonFiles", func(t *testing.T) {
		viper.Reset()

		// Test with key/value pair
		cmd := setupTestCommand(true)
		cmd.SetArgs([]string{"-c", "DataDir=testdir"})
		err := cmd.Execute()
		assert.Error(t, err, "Write mode should fail with key/value pair")

		// Test with non-file argument
		cmd = setupTestCommand(true)
		cmd.SetArgs([]string{"-c", "not_a_file"})
		err = cmd.Execute()
		assert.Error(t, err, "Write mode should fail with non-file argument")

		// Test with multiple key/value pairs
		cmd = setupTestCommand(true)
		cmd.SetArgs([]string{"-c", "DataDir=testdir", "-c", "API.Host=localhost"})
		err = cmd.Execute()
		assert.Error(t, err, "Write mode should fail with multiple key/value pairs")

		// Test with mix of file and key/value pair
		configFile, err := createTempConfig("")
		require.NoError(t, err)
		defer os.Remove(configFile)

		cmd = setupTestCommand(true)
		cmd.SetArgs([]string{"-c", configFile, "-c", "DataDir=testdir"})
		err = cmd.Execute()
		assert.Error(t, err, "Write mode should fail with mix of file and key/value pair")
	})
}
