//go:build unit || !integration

package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestConfigDataDirPath(t *testing.T) {
	workingDir, err := os.Getwd()
	require.NoError(t, err)
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	testCases := []struct {
		name          string
		dataDir       string
		setMethod     string // "cli", "config", "env", "file", "default"
		isValid       bool
		expected      string
		errorContains string
	}{
		// CLI flag cases
		{
			name:      "CLI: Valid relative path",
			dataDir:   "relative/path",
			setMethod: "cli",
			isValid:   true,
			expected:  filepath.Join(workingDir, "relative/path"),
		},
		{
			name:      "CLI: Valid tilde path",
			dataDir:   "~/some/path",
			setMethod: "cli",
			isValid:   true,
			expected:  filepath.Join(homeDir, "some/path"),
		},
		{
			name:      "CLI: Valid absolute path",
			dataDir:   "/absolute/path",
			setMethod: "cli",
			isValid:   true,
			expected:  "/absolute/path",
		},
		// Config value cases
		{
			name:      "Config: Valid relative path",
			dataDir:   "relative/path",
			setMethod: "config",
			isValid:   true,
			expected:  filepath.Join(workingDir, "relative/path"),
		},
		{
			name:      "Config: Valid tilde path",
			dataDir:   "~/some/path",
			setMethod: "config",
			isValid:   true,
			expected:  filepath.Join(homeDir, "some/path"),
		},
		{
			name:      "Config: Valid absolute path",
			dataDir:   "/absolute/path",
			setMethod: "config",
			isValid:   true,
			expected:  "/absolute/path",
		},
		// Environment variable cases
		{
			name:          "Env: Invalid relative path",
			dataDir:       "relative/path",
			setMethod:     "env",
			isValid:       false,
			errorContains: "not an absolute path",
		},
		{
			name:      "Env: Valid tilde path",
			dataDir:   "~/some/path",
			setMethod: "env",
			isValid:   true,
			expected:  filepath.Join(homeDir, "some/path"),
		},
		{
			name:      "Env: Valid absolute path",
			dataDir:   "/absolute/path",
			setMethod: "env",
			isValid:   true,
			expected:  "/absolute/path",
		},
		// Config file cases
		{
			name:          "File: Invalid relative path",
			dataDir:       "relative/path",
			setMethod:     "file",
			isValid:       false,
			errorContains: "not an absolute path",
		},
		{
			name:      "File: Valid tilde path",
			dataDir:   "~/some/path",
			setMethod: "file",
			isValid:   true,
			expected:  filepath.Join(homeDir, "some/path"),
		},
		{
			name:      "File: Valid absolute path",
			dataDir:   "/absolute/path",
			setMethod: "file",
			isValid:   true,
			expected:  "/absolute/path",
		},
		// Default value cases
		{
			name:          "Default: Invalid relative path",
			dataDir:       "relative/path",
			setMethod:     "default",
			isValid:       false,
			errorContains: "not an absolute path",
		},
		{
			name:      "Default: Valid tilde path",
			dataDir:   "~/some/path",
			setMethod: "default",
			isValid:   true,
			expected:  filepath.Join(homeDir, "some/path"),
		},
		{
			name:      "Default: Valid absolute path",
			dataDir:   "/absolute/path",
			setMethod: "default",
			isValid:   true,
			expected:  "/absolute/path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg *config.Config

			switch tc.setMethod {
			case "cli":
				flags := make(map[string][]*pflag.Flag)
				flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
				flagSet.String(types.DataDirKey, tc.dataDir, "data directory")
				flag := flagSet.Lookup(types.DataDirKey)
				flag.Changed = true
				flags[types.DataDirKey] = []*pflag.Flag{flag}
				cfg, err = config.New(config.WithFlags(flags))
			case "config":
				cfg, err = config.New(config.WithValues(map[string]interface{}{types.DataDirKey: tc.dataDir}))
			case "env":
				t.Setenv("BACALHAU_DATADIR", tc.dataDir)
				cfg, err = config.New(config.WithEnvironmentVariables(map[string][]string{
					types.DataDirKey: {"BACALHAU_DATADIR"},
				}))
			case "file":
				tempFile, innerErr := os.CreateTemp("", "config*.yaml")
				require.NoError(t, innerErr)
				defer os.Remove(tempFile.Name())
				_, innerErr = tempFile.WriteString(fmt.Sprintf("datadir: %s\n", tc.dataDir))
				require.NoError(t, innerErr)
				tempFile.Close()
				cfg, err = config.New(config.WithPaths(tempFile.Name()))
			case "default":
				cfg, err = config.New(config.WithDefault(types.Bacalhau{DataDir: tc.dataDir}))
			}

			if tc.isValid {
				require.NoError(t, err)
				var actual types.Bacalhau
				assert.NoError(t, cfg.Unmarshal(&actual))
				assert.Equal(t, tc.expected, actual.DataDir)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			}
		})
	}
}

func TestConfigWithValueOverrides(t *testing.T) {
	overrideRepo := "/overrideRepo"
	overrideName := "puuid"
	overrideClientAddress := "1.1.1.1"
	overrideClientPort := 1234

	defaultConfig := types.Bacalhau{
		DataDir: "/defaultRepo",
		API: types.API{
			Host: "0.0.0.0",
			Port: 1234,
		},
		Logging: types.Logging{
			Level: "info",
		},
	}
	overrideValues := map[string]any{
		"datadir":      overrideRepo,
		"nameprovider": overrideName,
		"api.host":     overrideClientAddress,
		"api.port":     overrideClientPort,
	}

	cfg, err := config.New(
		config.WithDefault(defaultConfig),
		config.WithValues(overrideValues),
	)
	require.NoError(t, err)

	var actual types.Bacalhau
	err = cfg.Unmarshal(&actual)
	require.NoError(t, err)

	assert.Equal(t, overrideRepo, actual.DataDir)
	assert.Equal(t, overrideClientAddress, actual.API.Host)
	assert.Equal(t, overrideClientPort, actual.API.Port)
	assert.Empty(t, actual.Orchestrator)
	assert.Empty(t, actual.Compute)
}

type TestConfig struct {
	StringValue string
	IntValue    int
	BoolValue   bool
	DataDir     string
}

func TestNew(t *testing.T) {
	t.Run("Default Configuration", func(t *testing.T) {
		cfg, err := config.New(config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    42,
			BoolValue:   true,
			DataDir:     "/data/dir",
		}))
		require.NoError(t, err)

		var testCfg TestConfig
		err = cfg.Unmarshal(&testCfg)
		require.NoError(t, err)

		assert.Equal(t, "default", testCfg.StringValue)
		assert.Equal(t, 42, testCfg.IntValue)
		assert.True(t, testCfg.BoolValue)
	})

	t.Run("With Configuration File", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "config*.yaml")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString(`
stringValue: "from_file"
intValue: 100
boolValue: false
datadir:   "/data/dir"
`)
		require.NoError(t, err)
		tempFile.Close()

		cfg, err := config.New(
			config.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   true,
			}),
			config.WithPaths(tempFile.Name()),
		)
		require.NoError(t, err)

		var testCfg TestConfig
		err = cfg.Unmarshal(&testCfg)
		require.NoError(t, err)

		assert.Equal(t, "from_file", testCfg.StringValue)
		assert.Equal(t, 100, testCfg.IntValue)
		assert.False(t, testCfg.BoolValue)
	})

	t.Run("With Flags", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("stringValue", "default", "test string value")
		flagSet.Int("intValue", 42, "test int value")
		flagSet.Bool("boolValue", true, "test bool value")

		err := flagSet.Parse([]string{"--stringValue=from_flag", "--intValue=200", "--boolValue=false"})
		require.NoError(t, err)

		flags := make(map[string][]*pflag.Flag)
		flagSet.VisitAll(func(f *pflag.Flag) {
			flags[f.Name] = []*pflag.Flag{f}
		})

		cfg, err := config.New(
			config.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   true,
				DataDir:     "/data/dir",
			}),
			config.WithFlags(flags),
		)
		require.NoError(t, err)

		var testCfg TestConfig
		err = cfg.Unmarshal(&testCfg)
		require.NoError(t, err)

		assert.Equal(t, "from_flag", testCfg.StringValue)
		assert.Equal(t, 200, testCfg.IntValue)
		assert.False(t, testCfg.BoolValue)
	})

	t.Run("With Environment Variables", func(t *testing.T) {
		t.Setenv("BACALHAU_STRING_VALUE", "from_env")
		t.Setenv("BACALHAU_INT_VALUE", "300")
		t.Setenv("BACALHAU_BOOL_VALUE", "true")

		envVars := map[string][]string{
			"stringValue": {"BACALHAU_STRING_VALUE"},
			"intValue":    {"BACALHAU_INT_VALUE"},
			"boolValue":   {"BACALHAU_BOOL_VALUE"},
		}

		cfg, err := config.New(
			config.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   false,
				DataDir:     "/data/dir",
			}),
			config.WithEnvironmentVariables(envVars),
		)
		require.NoError(t, err)

		var testCfg TestConfig
		err = cfg.Unmarshal(&testCfg)
		require.NoError(t, err)

		assert.Equal(t, "from_env", testCfg.StringValue)
		assert.Equal(t, 300, testCfg.IntValue)
		assert.True(t, testCfg.BoolValue)
	})

	t.Run("With Values", func(t *testing.T) {
		values := map[string]interface{}{
			"stringValue": "from_values",
			"intValue":    400,
			"boolValue":   false,
			"datadir":     "/data/dir",
		}

		cfg, err := config.New(
			config.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   true,
			}),
			config.WithValues(values),
		)
		require.NoError(t, err)

		var testCfg TestConfig
		err = cfg.Unmarshal(&testCfg)
		require.NoError(t, err)

		assert.Equal(t, "from_values", testCfg.StringValue)
		assert.Equal(t, 400, testCfg.IntValue)
		assert.False(t, testCfg.BoolValue)
	})
}

func TestLoad(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(`
stringValue: "loaded"
intValue: 500
boolValue: true
`)
	require.NoError(t, err)
	tempFile.Close()

	cfg, err := config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    42,
			BoolValue:   false,
			DataDir:     "/data/dir",
		}),
	)
	require.NoError(t, err)

	err = cfg.Load(tempFile.Name())
	require.NoError(t, err)

	var testCfg TestConfig
	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "loaded", testCfg.StringValue)
	assert.Equal(t, 500, testCfg.IntValue)
	assert.True(t, testCfg.BoolValue)
}

func TestMerge(t *testing.T) {
	tempFile1, err := os.CreateTemp("", "config1*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile1.Name())

	_, err = tempFile1.WriteString(`
stringValue: "first"
intValue: 100
datadir: "/dir1"
`)
	require.NoError(t, err)
	tempFile1.Close()

	tempFile2, err := os.CreateTemp("", "config2*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile2.Name())

	_, err = tempFile2.WriteString(`
intValue: 200
boolValue: true
datadir: "/dir2"
`)
	require.NoError(t, err)
	tempFile2.Close()

	cfg, err := config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    42,
			BoolValue:   false,
		}),
		config.WithPaths(tempFile1.Name(), tempFile2.Name()),
	)
	require.NoError(t, err)

	var testCfg TestConfig
	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "first", testCfg.StringValue)
	assert.Equal(t, 200, testCfg.IntValue)
	assert.True(t, testCfg.BoolValue)
	assert.Equal(t, "/dir2", testCfg.DataDir)
}

func TestConfigurationPrecedence(t *testing.T) {
	// Create a temporary config file
	tempFile, err := os.CreateTemp("", "config*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(`
stringValue: "from_file"
intValue: 100
boolValue: false
`)
	require.NoError(t, err)
	tempFile.Close()

	// Set up environment variables
	t.Setenv("BACALHAU_STRING_VALUE", "from_env")
	t.Setenv("BACALHAU_INT_VALUE", "200")
	t.Setenv("BACALHAU_BOOL_VALUE", "true")

	// Set up explicit values
	values := map[string]interface{}{
		"stringValue": "from_values",
		"intValue":    400,
		"boolValue":   true,
	}

	// Create the configuration
	cfg, err := config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   false,
			DataDir:     "/data/dir",
		}),
		config.WithPaths(tempFile.Name()),
		config.WithEnvironmentVariables(map[string][]string{
			"stringValue": {"BACALHAU_STRING_VALUE"},
			"intValue":    {"BACALHAU_INT_VALUE"},
			"boolValue":   {"BACALHAU_BOOL_VALUE"},
		}),
		config.WithValues(values),
	)
	require.NoError(t, err)

	var testCfg TestConfig
	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	// Assert the final configuration values
	assert.Equal(t, "from_values", testCfg.StringValue, "StringValue should be overridden by explicit values")
	assert.Equal(t, 400, testCfg.IntValue, "IntValue should be overridden by explicit values")
	assert.True(t, testCfg.BoolValue, "BoolValue should be overridden by explicit values")

	// Now, let's remove the explicit values and check the precedence of flags

	// Set up flags
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("stringValue", "", "test string value")
	flagSet.Int("intValue", 0, "test int value")
	flagSet.Bool("boolValue", false, "test bool value")

	err = flagSet.Parse([]string{"--stringValue=from_flag", "--intValue=300", "--boolValue=false"})
	require.NoError(t, err)

	flags := make(map[string][]*pflag.Flag)
	flagSet.VisitAll(func(f *pflag.Flag) {
		flags[f.Name] = []*pflag.Flag{f}
	})

	cfg, err = config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   false,
			DataDir:     "/data/dir",
		}),
		config.WithPaths(tempFile.Name()),
		config.WithEnvironmentVariables(map[string][]string{
			"stringValue": {"BACALHAU_STRING_VALUE"},
			"intValue":    {"BACALHAU_INT_VALUE"},
			"boolValue":   {"BACALHAU_BOOL_VALUE"},
		}),
		config.WithFlags(flags),
	)
	require.NoError(t, err)

	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "from_flag", testCfg.StringValue, "StringValue should be overridden by flags")
	assert.Equal(t, 300, testCfg.IntValue, "IntValue should be overridden by flags")
	assert.False(t, testCfg.BoolValue, "BoolValue should be overridden by flags")

	// Remove flags and check precedence of environment variables
	cfg, err = config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   false,
			DataDir:     "/data/dir",
		}),
		config.WithPaths(tempFile.Name()),
		config.WithEnvironmentVariables(map[string][]string{
			"stringValue": {"BACALHAU_STRING_VALUE"},
			"intValue":    {"BACALHAU_INT_VALUE"},
			"boolValue":   {"BACALHAU_BOOL_VALUE"},
		}),
	)
	require.NoError(t, err)

	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "from_env", testCfg.StringValue, "StringValue should be overridden by environment variables")
	assert.Equal(t, 200, testCfg.IntValue, "IntValue should be overridden by environment variables")
	assert.True(t, testCfg.BoolValue, "BoolValue should be overridden by environment variables")

	// Remove environment variables and check precedence of configuration file
	os.Unsetenv("BACALHAU_STRING_VALUE")
	os.Unsetenv("BACALHAU_INT_VALUE")
	os.Unsetenv("BACALHAU_BOOL_VALUE")

	cfg, err = config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   true,
			DataDir:     "/data/dir",
		}),
		config.WithPaths(tempFile.Name()),
	)
	require.NoError(t, err)

	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "from_file", testCfg.StringValue, "StringValue should be overridden by configuration file")
	assert.Equal(t, 100, testCfg.IntValue, "IntValue should be overridden by configuration file")
	assert.False(t, testCfg.BoolValue, "BoolValue should be overridden by configuration file")

	// Finally, check default values
	cfg, err = config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   true,
			DataDir:     "/data/dir",
		}),
	)
	require.NoError(t, err)

	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "default", testCfg.StringValue, "StringValue should be set to default")
	assert.Equal(t, 50, testCfg.IntValue, "IntValue should be set to default")
	assert.True(t, testCfg.BoolValue, "BoolValue should be set to default")
}

func TestFlagConfigConflicts(t *testing.T) {
	t.Run("No Conflicts", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("string-value", "test", "string value")
		flagSet.Int("int-value", 42, "int value")

		flags := make(map[string][]*pflag.Flag)
		flagSet.VisitAll(func(f *pflag.Flag) {
			flags[f.Name] = append(flags[f.Name], f)
		})

		values := map[string]any{
			"boolValue": true,
			"datadir":   "/data/dir",
		}

		cfg, err := config.New(
			config.WithDefault(TestConfig{}),
			config.WithFlags(flags),
			config.WithValues(values),
		)

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("With Conflicts", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("string-value", "test", "string value")
		flagSet.Int("int-value", 42, "int value")

		err := flagSet.Set("string-value", "flag-value")
		require.NoError(t, err)

		flags := make(map[string][]*pflag.Flag)
		flagSet.VisitAll(func(f *pflag.Flag) {
			flags[f.Name] = append(flags[f.Name], f)
		})

		values := map[string]any{
			"string-value": "conflict",
			"boolValue":    true,
		}

		cfg, err := config.New(
			config.WithDefault(TestConfig{}),
			config.WithFlags(flags),
			config.WithValues(values),
		)

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "flag: --string-value and config flag key \"string-value\" cannot both be provided")
	})

	t.Run("Multiple Flags for Same Key", func(t *testing.T) {
		flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
		flagSet.String("string-value", "test1", "string value 1")
		flagSet.String("str-value", "test2", "string value 2")

		err := flagSet.Set("str-value", "flag-value")
		require.NoError(t, err)

		flags := make(map[string][]*pflag.Flag)
		flagSet.VisitAll(func(f *pflag.Flag) {
			flags["stringValue"] = append(flags["stringValue"], f)
		})

		values := map[string]any{
			"stringValue": "conflict",
		}

		cfg, err := config.New(
			config.WithDefault(TestConfig{}),
			config.WithFlags(flags),
			config.WithValues(values),
		)

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "flag: --str-value and config flag key \"stringValue\" cannot both be provided")
	})
}

type LargeConfig struct {
	Database struct {
		Host     string
		Port     int
		Username string
		Password string
	}
	Server struct {
		Host string
		Port int
	}
	Logging struct {
		Level  string
		Format string
	}
}

func (c LargeConfig) Validate() error {
	return nil
}

type DatabaseConfig struct {
	Database struct {
		Host     string
		Port     int
		Username string
		Password string
	}
}

func (c DatabaseConfig) Validate() error {
	return nil
}

type ServerLoggingConfig struct {
	Server struct {
		Host string
		Port int
	}
	Logging struct {
		Level string
	}
}

func (c ServerLoggingConfig) Validate() error {
	return nil
}

type ExtendedServerConfig struct {
	Server struct {
		Host    string
		Port    int
		Timeout int // This field is not in the original config
	}
}

func (c ExtendedServerConfig) Validate() error {
	return nil
}

// TestUnmarshalSubset tests that config.Unmarshal works correctly with subsets of a larger configuration
func TestUnmarshalSubset(t *testing.T) {

	// Create a temporary config file with all settings
	tempFile, err := os.CreateTemp("", "large_config*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.WriteString(`
database:
  host: "localhost"
  port: 5432
  username: "user"
  password: "password"
server:
  host: "0.0.0.0"
  port: 8080
logging:
  level: "info"
  format: "json"
`)
	require.NoError(t, err)
	tempFile.Close()

	// Create the configuration
	cfg, err := config.New(
		config.WithPaths(tempFile.Name()),
	)
	require.NoError(t, err)

	// Test unmarshaling the entire configuration
	var fullConfig LargeConfig
	err = cfg.Unmarshal(&fullConfig)
	require.NoError(t, err)
	require.NoError(t, fullConfig.Validate())

	assert.Equal(t, "localhost", fullConfig.Database.Host)
	assert.Equal(t, 5432, fullConfig.Database.Port)
	assert.Equal(t, "user", fullConfig.Database.Username)
	assert.Equal(t, "password", fullConfig.Database.Password)
	assert.Equal(t, "0.0.0.0", fullConfig.Server.Host)
	assert.Equal(t, 8080, fullConfig.Server.Port)
	assert.Equal(t, "info", fullConfig.Logging.Level)
	assert.Equal(t, "json", fullConfig.Logging.Format)

	// dbConfig is a subset, ensure we can unmarshal into it.
	var dbConfig DatabaseConfig
	err = cfg.Unmarshal(&dbConfig)
	require.NoError(t, err)
	require.NoError(t, dbConfig.Validate())

	assert.Equal(t, "localhost", dbConfig.Database.Host)
	assert.Equal(t, 5432, dbConfig.Database.Port)
	assert.Equal(t, "user", dbConfig.Database.Username)
	assert.Equal(t, "password", dbConfig.Database.Password)

	var slConfig ServerLoggingConfig
	err = cfg.Unmarshal(&slConfig)
	require.NoError(t, err)
	require.NoError(t, slConfig.Validate())

	assert.Equal(t, "0.0.0.0", slConfig.Server.Host)
	assert.Equal(t, 8080, slConfig.Server.Port)
	assert.Equal(t, "info", slConfig.Logging.Level)

	var extConfig ExtendedServerConfig
	err = cfg.Unmarshal(&extConfig)
	require.NoError(t, err)
	require.NoError(t, extConfig.Validate())

	assert.Equal(t, "0.0.0.0", extConfig.Server.Host)
	assert.Equal(t, 8080, extConfig.Server.Port)
	assert.Zero(t, extConfig.Server.Timeout, "Timeout should be zero as it's not in the original config")
}

func TestDefaultConfigPath(t *testing.T) {
	// Create a temporary directory to act as the data directory
	tempDir, err := os.MkdirTemp("", "bacalhau-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a config file in the temporary directory
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(`
stringValue: "from_file"
intValue: 100
boolValue: true
`), 0644)
	require.NoError(t, err)

	// Set environment variable
	t.Setenv("BACALHAU_INT_VALUE", "200")

	// Create the configuration with all sources
	cfg, err := config.New(
		config.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   false,
		}),
		config.WithValues(map[string]interface{}{
			types.DataDirKey: tempDir,
			"boolValue":      false,
		}),
		config.WithEnvironmentVariables(map[string][]string{
			"intValue": {"BACALHAU_INT_VALUE"},
		}),
	)
	require.NoError(t, err)

	var testCfg TestConfig
	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	// Check precedence and file reading
	assert.Equal(t, 200, testCfg.IntValue, "Environment variable should take precedence over file")
	assert.False(t, testCfg.BoolValue, "Explicit config value should take precedence over file")
	assert.Equal(t, "from_file", testCfg.StringValue, "File value should be read from file")

	// Ensure the default config file was used
	assert.Equal(t, configPath, cfg.ConfigFileUsed(), "Default config file should be used")
}
