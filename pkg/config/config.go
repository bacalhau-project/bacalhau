package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
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
	// viper instance for holding user provided configuration
	user *viper.Viper
	// viper instance for holding system specific configuration
	system *viper.Viper
	// the default configuration values to initialize with
	defaultCfg types.BacalhauConfig
}

type Option = func(s *Config)

func WithDefaultConfig(cfg types.BacalhauConfig) Option {
	return func(c *Config) {
		c.defaultCfg = cfg
	}
}

func New(opts ...Option) *Config {
	c := &Config{
		user:       viper.New(),
		system:     viper.New(),
		defaultCfg: configenv.Production,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.user.SetEnvPrefix(environmentVariablePrefix)
	c.user.SetTypeByDefaultValue(inferConfigTypes)
	c.user.AutomaticEnv()
	c.user.SetEnvKeyReplacer(environmentVariableReplace)
	c.setDefault(c.defaultCfg)
	return c
}

func (c *Config) RepoPath() (string, error) {
	repoPath := c.system.GetString("repo")
	if repoPath == "" {

		return "", fmt.Errorf("repo path not configured")
	}
	return repoPath, nil
}

func (c *Config) Load(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// if the config file doesn't exist then we obviously cannot load it
		return fmt.Errorf("config file not found at at path: %q", path)
	} else if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	c.user.SetConfigFile(path)
	if err := c.user.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config file")
	}
	return nil
}

// setDefault sets the default value for the configuration.
// Default only used when no value is provided by the user via an explicit call to Set, flag, config file or ENV.
func (c *Config) setDefault(config types.BacalhauConfig) {
	types.SetDefaults(config, types.WithViper(c.user))
}

func (c *Config) Current() (types.BacalhauConfig, error) {
	out := new(types.BacalhauConfig)
	if err := c.user.Unmarshal(&out, DecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}
	return *out, nil

}

func (c *Config) User() *viper.Viper {
	return c.user
}

func (c *Config) System() *viper.Viper {
	return c.system
}

// Set sets the configuration value.
// This value won't be persisted in the config file.
// Will be used instead of values obtained via flags, config file, ENV, default.
func (c *Config) Set(key string, value interface{}) {
	c.user.Set(key, value)
}

func (c *Config) SetIfAbsent(key string, value interface{}) {
	thing := c.user.Get(key)
	_ = thing
	if !c.user.IsSet(key) || reflect.ValueOf(c.user.Get(key)).IsZero() {
		c.Set(key, value)
	}
}

func (c *Config) Get(key string) interface{} {
	return c.user.Get(key)
}

func (c *Config) GetString(key string) (string, bool) {
	out := c.user.GetString(key)
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
	return unmarshalCompositeKey(c.user, key, cfg)
}
