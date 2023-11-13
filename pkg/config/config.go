package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

const (
	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true
	automaticEnvVar           = true

	// user key files
	Libp2pPrivateKeyFileName = "libp2p_private_key"
	UserPrivateKeyFileName   = "user_id.pem"

	// compute paths
	ComputeStoragesPath = "executor_storages"
	PluginsPath         = "plugins"

	// requester paths
	AutoCertCachePath = "autocert-cache"

	// update check paths
	UpdateCheckStatePath = "update.json"
)

var (
	environmentVariableReplace = strings.NewReplacer(".", "_")
	configDecoderHook          = viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())
)

const (
	configType = "yaml"
	configName = "config"
)

func Init(path string) (types.BacalhauConfig, error) {
	// derive the default config for the specified environment.
	defaultConfig := ForEnvironment()

	// set default values for path dependent config.
	defaultConfig.User.KeyPath = filepath.Join(path, UserPrivateKeyFileName)
	defaultConfig.User.Libp2pKeyPath = filepath.Join(path, Libp2pPrivateKeyFileName)
	defaultConfig.Node.ExecutorPluginPath = filepath.Join(path, PluginsPath)
	defaultConfig.Node.ComputeStoragePath = filepath.Join(path, ComputeStoragesPath)
	defaultConfig.Node.ServerAPI.TLS.AutoCertCachePath = filepath.Join(path, AutoCertCachePath)
	defaultConfig.Update.CheckStatePath = filepath.Join(path, UpdateCheckStatePath)

	// initialize the configuration with default values.
	return initConfig(path, WithDefaultConfig(defaultConfig))
}

func Load(path string) (types.BacalhauConfig, error) {
	// derive the default config for the specified environment.
	defaultConfig := ForEnvironment()

	// set default values for path dependent config.
	defaultConfig.User.KeyPath = filepath.Join(path, UserPrivateKeyFileName)
	defaultConfig.User.Libp2pKeyPath = filepath.Join(path, Libp2pPrivateKeyFileName)
	defaultConfig.Node.ExecutorPluginPath = filepath.Join(path, PluginsPath)
	defaultConfig.Node.ComputeStoragePath = filepath.Join(path, ComputeStoragesPath)
	defaultConfig.Node.ServerAPI.TLS.AutoCertCachePath = filepath.Join(path, AutoCertCachePath)
	defaultConfig.Update.CheckStatePath = filepath.Join(path, UpdateCheckStatePath)

	return initConfig(path, WithDefaultConfig(defaultConfig), WithFileHandler(ReadConfigHandler))
}

type Params struct {
	FileName      string
	FileType      string
	FileHandler   func(fileName string) error
	DefaultConfig types.BacalhauConfig
}

func initConfig(path string, opts ...Option) (types.BacalhauConfig, error) {
	params := &Params{
		FileName:      configName,
		FileType:      configType,
		FileHandler:   NoopConfigHandler,
		DefaultConfig: ForEnvironment(),
	}

	for _, opt := range opts {
		opt(params)
	}

	viper.AddConfigPath(path)
	viper.SetConfigName(params.FileName)
	viper.SetConfigType(params.FileType)
	viper.SetEnvPrefix(environmentVariablePrefix)
	viper.SetTypeByDefaultValue(inferConfigTypes)
	viper.SetEnvKeyReplacer(environmentVariableReplace)
	if err := SetDefault(params.DefaultConfig); err != nil {
		return types.BacalhauConfig{}, nil
	}

	if err := params.FileHandler(filepath.Join(path, fmt.Sprintf("%s.%s", params.FileName, params.FileType))); err != nil {
		return types.BacalhauConfig{}, err
	}

	if automaticEnvVar {
		viper.AutomaticEnv()
	}

	var out types.BacalhauConfig
	if err := viper.Unmarshal(&out, configDecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}

	return out, nil
}

// Reset clears all configuration, useful for testing.
func Reset() {
	viper.Reset()
}

// Getenv wraps os.Getenv and retrieves the value of the environment variable named by the config key.
// It returns the value, which will be empty if the variable is not present.
func Getenv(key string) string {
	return os.Getenv(KeyAsEnvVar(key))
}

// KeyAsEnvVar returns the environment variable corresponding to a config key
func KeyAsEnvVar(key string) string {
	return strings.ToUpper(
		fmt.Sprintf("%s_%s", environmentVariablePrefix, environmentVariableReplace.Replace(key)),
	)
}
