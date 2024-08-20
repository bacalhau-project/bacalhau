//go:build unit || !integration

package configv2_test

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/configv2"
	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

func TestConfigWithDefaults(t *testing.T) {
	expected := configv2.Default

	cfg, err := configv2.New(
		configv2.WithDefault(expected),
	)
	require.NoError(t, err)

	var actual types.Bacalhau
	err = cfg.Unmarshal(&actual)
	require.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestConfigWithValueOverrides(t *testing.T) {
	overrideRepo := "overrideRepo"
	overrideName := "overrideName"
	overrideClientAddress := "overrideAddress"

	defaultConfig := types.Bacalhau{
		DataDir: "defaultRepo",
		API: types.API{
			Address: "defaultAddress",
		},
	}
	overrideValues := map[string]any{
		"datadir":      overrideRepo,
		"nameprovider": overrideName,
		"api.address":  overrideClientAddress,
	}

	cfg, err := configv2.New(
		configv2.WithDefault(defaultConfig),
		configv2.WithValues(overrideValues),
	)
	require.NoError(t, err)

	var actual types.Bacalhau
	err = cfg.Unmarshal(&actual)
	require.NoError(t, err)

	assert.Equal(t, overrideRepo, actual.DataDir)
	assert.Equal(t, overrideClientAddress, actual.API.Address)
	assert.Empty(t, actual.Orchestrator)
	assert.Empty(t, actual.Compute)
}

type TestConfig struct {
	StringValue string
	IntValue    int
	BoolValue   bool
}

func (c TestConfig) Validate() error {
	return nil
}

func TestNew(t *testing.T) {
	t.Run("Default Configuration", func(t *testing.T) {
		cfg, err := configv2.New(configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    42,
			BoolValue:   true,
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
`)
		require.NoError(t, err)
		tempFile.Close()

		cfg, err := configv2.New(
			configv2.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   true,
			}),
			configv2.WithPaths(tempFile.Name()),
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

		flags := make(map[string]*pflag.Flag)
		flagSet.VisitAll(func(f *pflag.Flag) {
			flags[f.Name] = f
		})

		cfg, err := configv2.New(
			configv2.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   true,
			}),
			configv2.WithFlags(flags),
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
		os.Setenv("BACALHAU_STRING_VALUE", "from_env")
		os.Setenv("BACALHAU_INT_VALUE", "300")
		os.Setenv("BACALHAU_BOOL_VALUE", "true")
		defer func() {
			os.Unsetenv("BACALHAU_STRING_VALUE")
			os.Unsetenv("BACALHAU_INT_VALUE")
			os.Unsetenv("BACALHAU_BOOL_VALUE")
		}()

		envVars := map[string][]string{
			"stringValue": {"BACALHAU_STRING_VALUE"},
			"intValue":    {"BACALHAU_INT_VALUE"},
			"boolValue":   {"BACALHAU_BOOL_VALUE"},
		}

		cfg, err := configv2.New(
			configv2.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   false,
			}),
			configv2.WithEnvironmentVariables(envVars),
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
		}

		cfg, err := configv2.New(
			configv2.WithDefault(TestConfig{
				StringValue: "default",
				IntValue:    42,
				BoolValue:   true,
			}),
			configv2.WithValues(values),
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

	cfg, err := configv2.New(
		configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    42,
			BoolValue:   false,
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
`)
	require.NoError(t, err)
	tempFile1.Close()

	tempFile2, err := os.CreateTemp("", "config2*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile2.Name())

	_, err = tempFile2.WriteString(`
intValue: 200
boolValue: true
`)
	require.NoError(t, err)
	tempFile2.Close()

	cfg, err := configv2.New(
		configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    42,
			BoolValue:   false,
		}),
		configv2.WithPaths(tempFile1.Name()),
	)
	require.NoError(t, err)

	err = cfg.Merge(tempFile2.Name())
	require.NoError(t, err)

	var testCfg TestConfig
	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "first", testCfg.StringValue)
	assert.Equal(t, 200, testCfg.IntValue)
	assert.True(t, testCfg.BoolValue)
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
	os.Setenv("BACALHAU_STRING_VALUE", "from_env")
	os.Setenv("BACALHAU_INT_VALUE", "200")
	os.Setenv("BACALHAU_BOOL_VALUE", "true")
	defer func() {
		os.Unsetenv("BACALHAU_STRING_VALUE")
		os.Unsetenv("BACALHAU_INT_VALUE")
		os.Unsetenv("BACALHAU_BOOL_VALUE")
	}()

	// Set up flags
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("stringValue", "", "test string value")
	flagSet.Int("intValue", 0, "test int value")
	flagSet.Bool("boolValue", false, "test bool value")

	err = flagSet.Parse([]string{"--stringValue=from_flag", "--intValue=300", "--boolValue=false"})
	require.NoError(t, err)

	flags := make(map[string]*pflag.Flag)
	flagSet.VisitAll(func(f *pflag.Flag) {
		flags[f.Name] = f
	})

	// Set up explicit values
	values := map[string]interface{}{
		"stringValue": "from_values",
		"intValue":    400,
		"boolValue":   true,
	}

	// Create the configuration
	cfg, err := configv2.New(
		configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   false,
		}),
		configv2.WithPaths(tempFile.Name()),
		configv2.WithEnvironmentVariables(map[string][]string{
			"stringValue": {"BACALHAU_STRING_VALUE"},
			"intValue":    {"BACALHAU_INT_VALUE"},
			"boolValue":   {"BACALHAU_BOOL_VALUE"},
		}),
		configv2.WithFlags(flags),
		configv2.WithValues(values),
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
	cfg, err = configv2.New(
		configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   false,
		}),
		configv2.WithPaths(tempFile.Name()),
		configv2.WithEnvironmentVariables(map[string][]string{
			"stringValue": {"BACALHAU_STRING_VALUE"},
			"intValue":    {"BACALHAU_INT_VALUE"},
			"boolValue":   {"BACALHAU_BOOL_VALUE"},
		}),
		configv2.WithFlags(flags),
	)
	require.NoError(t, err)

	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "from_flag", testCfg.StringValue, "StringValue should be overridden by flags")
	assert.Equal(t, 300, testCfg.IntValue, "IntValue should be overridden by flags")
	assert.False(t, testCfg.BoolValue, "BoolValue should be overridden by flags")

	// Remove flags and check precedence of environment variables
	cfg, err = configv2.New(
		configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   false,
		}),
		configv2.WithPaths(tempFile.Name()),
		configv2.WithEnvironmentVariables(map[string][]string{
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

	cfg, err = configv2.New(
		configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   true,
		}),
		configv2.WithPaths(tempFile.Name()),
	)
	require.NoError(t, err)

	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "from_file", testCfg.StringValue, "StringValue should be overridden by configuration file")
	assert.Equal(t, 100, testCfg.IntValue, "IntValue should be overridden by configuration file")
	assert.False(t, testCfg.BoolValue, "BoolValue should be overridden by configuration file")

	// Finally, check default values
	cfg, err = configv2.New(
		configv2.WithDefault(TestConfig{
			StringValue: "default",
			IntValue:    50,
			BoolValue:   true,
		}),
	)
	require.NoError(t, err)

	err = cfg.Unmarshal(&testCfg)
	require.NoError(t, err)

	assert.Equal(t, "default", testCfg.StringValue, "StringValue should be set to default")
	assert.Equal(t, 50, testCfg.IntValue, "IntValue should be set to default")
	assert.True(t, testCfg.BoolValue, "BoolValue should be set to default")
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
	cfg, err := configv2.New(
		configv2.WithPaths(tempFile.Name()),
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
