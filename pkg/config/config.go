package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

type ReadWriter interface {
	Reader
	Writer
}

var _ Reader = (*config)(nil)

type Reader interface {
	Current() (types.BacalhauConfig, error)
}

var _ Writer = (*config)(nil)

type Writer interface {
	Load(path string) error
	Set(key string, value interface{})
	SetIfAbsent(key string, value interface{})
}

const (
	FileName = "config.yaml"

	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true

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

type config struct {
	// viper instance for holding user provided configuration
	v *viper.Viper
	// the default configuration values to initialize with
	defaultCfg types.BacalhauConfig
}

type Option = func(s *config)

func WithDefaultConfig(cfg types.BacalhauConfig) Option {
	return func(c *config) {
		c.defaultCfg = cfg
	}
}

func WithViper(v *viper.Viper) Option {
	return func(c *config) {
		c.v = v
	}
}

func New(opts ...Option) *config {
	c := &config{
		v:          viper.New(),
		defaultCfg: configenv.Production,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.v.SetEnvPrefix(environmentVariablePrefix)
	c.v.SetTypeByDefaultValue(inferConfigTypes)
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(environmentVariableReplace)
	c.setDefault(c.defaultCfg)
	return c
}

func (c *config) Load(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// if the config file doesn't exist then we obviously cannot load it
		return fmt.Errorf("config file not found at at path: %q: %w", path, err)
	} else if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	c.v.SetConfigFile(path)
	if err := c.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}
	return nil
}

func (c *config) Write(path string) error {
	return c.v.WriteConfigAs(path)
}

func (c *config) Current() (types.BacalhauConfig, error) {
	out := new(types.BacalhauConfig)
	if err := c.v.Unmarshal(&out, DecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}
	return *out, nil
}

// Set sets the configuration value.
// This value won't be persisted in the config file.
// Will be used instead of values obtained via flags, config file, ENV, default.
func (c *config) Set(key string, value interface{}) {
	c.v.Set(key, value)
}

func (c *config) SetIfAbsent(key string, value interface{}) {
	if !c.v.IsSet(key) || reflect.ValueOf(c.v.Get(key)).IsZero() {
		c.Set(key, value)
	}
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
/*
func (c *config) ForKey(key string, cfg interface{}) error {
	return unmarshalCompositeKey(c.v, key, cfg)
}
*/

// setDefault sets the default value for the configuration.
// Default only used when no value is provided by the user via an explicit call to Set, flag, config file or ENV.
func (c *config) setDefault(config types.BacalhauConfig) {
	types.SetDefaults(config, types.WithViper(c.v))
}

// WritePersistedConfigs will write certain values from the resolved config to the persisted config.
// These include fields for configurations that must not change between version updates, such as the
// execution store and job store paths, in case we change their default values in future updates.
func WritePersistedConfigs(configFilePath string, cfg types.BacalhauConfig) error {
	// a viper config instance that is only based on the config file.
	viperWriter := viper.New()
	viperWriter.SetTypeByDefaultValue(true)
	viperWriter.SetConfigFile(configFilePath)

	// read existing config if it exists.
	if err := viperWriter.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	var fileCfg types.BacalhauConfig
	if err := viperWriter.Unmarshal(&fileCfg, DecoderHook); err != nil {
		return err
	}

	// check if any of the values that we want to write are not set in the config file.
	var doWrite bool
	var logMessage strings.Builder
	set := func(key string, value interface{}) {
		viperWriter.Set(key, value)
		logMessage.WriteString(fmt.Sprintf("\n%s:\t%v", key, value))
		doWrite = true
	}
	emptyStoreConfig := types.JobStoreConfig{}
	if fileCfg.Node.Compute.ExecutionStore == emptyStoreConfig {
		set(types.NodeComputeExecutionStore, cfg.Node.Compute.ExecutionStore)
	}
	if fileCfg.Node.Requester.JobStore == emptyStoreConfig {
		set(types.NodeRequesterJobStore, cfg.Node.Requester.JobStore)
	}
	if fileCfg.Node.Name == "" && cfg.Node.Name != "" {
		set(types.NodeName, cfg.Node.Name)
	}
	if doWrite {
		log.Info().Msgf("Writing to config file %s:%s", configFilePath, logMessage.String())
		return viperWriter.WriteConfig()
	}
	return nil
}
