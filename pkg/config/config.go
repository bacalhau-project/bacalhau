package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

const (
	ConfigFileName = "config.yaml"
	ConfigFileMode = 0666

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
	NetworkTransportStore = "nats-store"

	// update check paths
	UpdateCheckStatePath = "update.json"

	// auth paths
	TokensPath = "tokens.json"
)

var (
	ComputeExecutionsStorePath = filepath.Join(ComputeStorePath, "executions.db")
	OrchestratorJobStorePath   = filepath.Join(OrchestratorStorePath, "jobs.db")

	environmentVariableReplace = strings.NewReplacer(".", "_")
	DecoderHook                = viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())
)

type Config struct {
	viper  *viper.Viper
	system *viper.Viper
}

func New() *Config {
	c := &Config{viper: viper.New(), system: viper.New()}
	return c
}

func (c *Config) RepoPath() (string, error) {
	repoPath := c.system.GetString("repo")
	if repoPath == "" {

		return "", fmt.Errorf("repo path not configured")
	}
	return repoPath, nil
}

func (c *Config) Init(path string) (types.BacalhauConfig, error) {
	// initialize the configuration with default values.
	return c.initConfig(path,
		WithDefaultConfig(getDefaultConfig(path)))

}

func (c *Config) Load(path string) (types.BacalhauConfig, error) {
	return c.initConfig(path,
		WithDefaultConfig(getDefaultConfig(path)),
		WithFileHandler(ReadConfigHandler),
	)
}

type params struct {
	FileName      string
	FileHandler   func(v *viper.Viper, fileName string) error
	DefaultConfig types.BacalhauConfig
}

func (c *Config) initConfig(path string, opts ...Option) (types.BacalhauConfig, error) {
	params := &params{
		FileName:      ConfigFileName,
		FileHandler:   NoopConfigHandler,
		DefaultConfig: ForEnvironment(),
	}

	for _, opt := range opts {
		opt(params)
	}

	configFile := filepath.Join(path, params.FileName)
	c.viper.SetConfigFile(configFile)
	c.viper.SetEnvPrefix(environmentVariablePrefix)
	c.viper.SetTypeByDefaultValue(inferConfigTypes)
	c.viper.SetEnvKeyReplacer(environmentVariableReplace)
	c.SetDefault(params.DefaultConfig)

	if err := params.FileHandler(c.viper, configFile); err != nil {
		return types.BacalhauConfig{}, err
	}

	if automaticEnvVar {
		c.viper.AutomaticEnv()
	}

	var out types.BacalhauConfig
	if err := c.viper.Unmarshal(&out, DecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}

	return out, nil
}

// Reset clears all configuration, useful for testing.
func (c *Config) Reset() {
	c.viper = viper.New()
}

// SetDefault sets the default value for the configuration.
// Default only used when no value is provided by the user via an explicit call to Set, flag, config file or ENV.
func (c *Config) SetDefault(config types.BacalhauConfig) {
	types.SetDefaults(config, types.WithViper(c.viper))
}

func (c *Config) Current() (types.BacalhauConfig, error) {
	out := new(types.BacalhauConfig)
	if err := c.viper.Unmarshal(&out, DecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}
	return *out, nil

}

func (c *Config) Viper() *viper.Viper {
	return c.viper
}

func (c *Config) System() *viper.Viper {
	return c.system
}

// SetValue sets the configuration value.
// This value won't be persisted in the config file.
// Will be used instead of values obtained via flags, config file, ENV, default.
func (c *Config) SetValue(key string, value interface{}) {
	c.viper.Set(key, value)
}

func (c *Config) GetString(key string) (string, bool) {
	out := c.viper.GetString(key)
	if out == "" {
		return out, false
	}
	return out, true
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
func (c *Config) ForKey(key string, cfg interface{}) error {
	return unmarshalCompositeKey(c.viper, key, cfg)
}
