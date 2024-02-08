package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
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
	ComputeStorePath    = "compute_store"
	PluginsPath         = "plugins"

	// orchestrator paths
	OrchestratorStorePath = "orchestrator_store"
	AutoCertCachePath     = "autocert-cache"

	// update check paths
	UpdateCheckStatePath = "update.json"

	// auth paths
	TokensPath = "tokens.json"
)

var (
	ComputeExecutionsStorePath = filepath.Join(ComputeStorePath, "executions.db")
	OrchestratorJobStorePath   = filepath.Join(OrchestratorStorePath, "jobs.db")
)

var (
	environmentVariableReplace = strings.NewReplacer(".", "_")
	DecoderHook                = viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())
)

const (
	ConfigFileName = "config.yaml"
	ConfigFileMode = 0666
)

func Init(path string) (types.BacalhauConfig, error) {
	// initialize the configuration with default values.
	return initConfig(path,
		WithDefaultConfig(getDefaultConfig(path)),
		WithPostConfigHandler(WritePersistedConfigs),
	)
}

func Load(path string) (types.BacalhauConfig, error) {
	return initConfig(path,
		WithDefaultConfig(getDefaultConfig(path)),
		WithFileHandler(ReadConfigHandler),
	)
}

func getDefaultConfig(path string) types.BacalhauConfig {
	// derive the default config for the specified environment.
	defaultConfig := ForEnvironment()

	// set default values for path dependent config.
	defaultConfig.User.KeyPath = filepath.Join(path, UserPrivateKeyFileName)
	defaultConfig.User.Libp2pKeyPath = filepath.Join(path, Libp2pPrivateKeyFileName)
	defaultConfig.Node.ExecutorPluginPath = filepath.Join(path, PluginsPath)
	defaultConfig.Node.ComputeStoragePath = filepath.Join(path, ComputeStoragesPath)
	defaultConfig.Node.Compute.ExecutionStore.Path = filepath.Join(path, ComputeExecutionsStorePath)
	defaultConfig.Node.Requester.JobStore.Path = filepath.Join(path, OrchestratorJobStorePath)
	defaultConfig.Update.CheckStatePath = filepath.Join(path, UpdateCheckStatePath)
	defaultConfig.Auth.TokensPath = filepath.Join(path, TokensPath)

	return defaultConfig
}

type Params struct {
	FileName          string
	FileHandler       func(fileName string) error
	PostConfigHandler func(fileName string, cfg types.BacalhauConfig) error
	DefaultConfig     types.BacalhauConfig
}

func initConfig(path string, opts ...Option) (types.BacalhauConfig, error) {
	params := &Params{
		FileName:          ConfigFileName,
		FileHandler:       NoopConfigHandler,
		PostConfigHandler: NoopPostConfigHandler,
		DefaultConfig:     ForEnvironment(),
	}

	for _, opt := range opts {
		opt(params)
	}

	configFile := filepath.Join(path, params.FileName)
	viper.SetConfigFile(configFile)
	viper.SetEnvPrefix(environmentVariablePrefix)
	viper.SetTypeByDefaultValue(inferConfigTypes)
	viper.SetEnvKeyReplacer(environmentVariableReplace)
	if err := SetDefault(params.DefaultConfig); err != nil {
		return types.BacalhauConfig{}, nil
	}

	if err := params.FileHandler(configFile); err != nil {
		return types.BacalhauConfig{}, err
	}

	if automaticEnvVar {
		viper.AutomaticEnv()
	}

	var out types.BacalhauConfig
	if err := viper.Unmarshal(&out, DecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}

	if err := params.PostConfigHandler(configFile, out); err != nil {
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
