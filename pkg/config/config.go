package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

const (
	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true
	automaticEnvVar           = true
)

var (
	environmentVariableReplace = strings.NewReplacer(".", "_")
	configDecoderHook          = viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())
)

func Init(defaultConfig *types.BacalhauConfig, path string, fileName, fileType string) (types.BacalhauConfig, error) {
	return initConfig(initParams{
		filePath:      path,
		fileName:      fileName,
		fileType:      fileType,
		fileHandler:   viper.SafeWriteConfig,
		defaultConfig: defaultConfig,
	})
}

func Load(path string, fileName, fileType string) (types.BacalhauConfig, error) {
	return initConfig(initParams{
		filePath:      path,
		fileName:      fileName,
		fileType:      fileType,
		fileHandler:   viper.ReadInConfig,
		defaultConfig: nil,
	})
}

func Get[T any](key string) (T, error) {
	raw := viper.Get(key)
	if raw == nil {
		return zeroValue[T](), fmt.Errorf("value not found for %s", key)
	}

	val, ok := raw.(T)
	if !ok {
		return zeroValue[T](), fmt.Errorf("value not of expected type, got: %T", raw)
	}

	return val, nil
}

func zeroValue[T any]() T {
	var zero T
	return zero
}

// Set sets the current configuration to `config`, useful for testing.
func Set(config types.BacalhauConfig) error {
	types.SetDefaults(config)
	return nil
}

// Reset clears all configuration, useful for testing.
func Reset() {
	viper.Reset()
}

// KeyAsEnvVar returns the environment variable corresponding to a config key
func KeyAsEnvVar(key string) string {
	return strings.ToUpper(
		fmt.Sprintf("%s_%s", environmentVariablePrefix, environmentVariableReplace.Replace(key)),
	)
}

// ForKey unmarshals configuration values associated with a given key into the provided cfg structure.
// It uses unmarshalCompositeKey internally to handle composite keys, ensuring values spread across
// nested sub-keys are correctly populated into the cfg structure.
//
// Parameters:
//   - key: The configuration key to retrieve values for.
//   - cfg: The structure into which the configuration values will be unmarshaled.
//
// Returns:
//   - An error if any occurred during unmarshaling; otherwise, nil.
func ForKey(key string, cfg interface{}) error {
	return unmarshalCompositeKey(key, cfg)
}

type initParams struct {
	filePath      string
	fileName      string
	fileType      string
	fileHandler   func() error
	defaultConfig *types.BacalhauConfig
}

func initConfig(params initParams) (types.BacalhauConfig, error) {
	viper.AddConfigPath(params.filePath)
	viper.SetConfigName(params.fileName)
	viper.SetConfigType(params.fileType)
	viper.SetEnvPrefix(environmentVariablePrefix)
	viper.SetTypeByDefaultValue(inferConfigTypes)
	viper.SetEnvKeyReplacer(environmentVariableReplace)
	if params.defaultConfig != nil {
		if err := Set(*params.defaultConfig); err != nil {
			return types.BacalhauConfig{}, nil
		}
	}
	if err := params.fileHandler(); err != nil {
		return types.BacalhauConfig{}, err
	}
	if automaticEnvVar {
		viper.AutomaticEnv()
	}

	var out types.BacalhauConfig
	if err := viper.Unmarshal(&out, configDecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()

	return out, nil
}
