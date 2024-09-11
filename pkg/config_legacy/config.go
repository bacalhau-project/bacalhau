package config_legacy

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config_legacy/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
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
	Set(key string, value interface{})
	SetIfAbsent(key string, value interface{})
}

const (
	FileName = "config.yaml"

	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true

	// user key files
	UserPrivateKeyFileName = "user_id.pem"

	// compute paths
	ComputeStoragesPath = "executor_storages"
	ComputeStorePath    = "compute_store"
	PluginsPath         = "plugins"

	// orchestrator paths
	OrchestratorStorePath = "orchestrator_store"
	AutoCertCachePath     = "autocert-cache"
	NetworkTransportStore = "nats-store"

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
	// viper instance for holding user provided configuration.
	base *viper.Viper
	// the default configuration values to initialize with.
	defaultCfg types.BacalhauConfig

	// paths to configuration files merged from [0] to [N]
	// e.g. file at index 1 overrides index 0, index 2 overrides index 1 and 0, etc.
	paths []string

	flags map[string]*pflag.Flag

	environmentVariables map[string][]string

	// values to inject into the config, taking highest precedence.
	values map[string]any
}

type Option = func(s *config)

// WithDefault sets the default config to be used when no values are provided.
func WithDefault(cfg types.BacalhauConfig) Option {
	return func(c *config) {
		c.defaultCfg = cfg
	}
}

// WithPaths sets paths to configuration files to be loaded
// paths to configuration files merged from [0] to [N]
// e.g. file at index 1 overrides index 0, index 2 overrides index 1 and 0, etc.
func WithPaths(path ...string) Option {
	return func(c *config) {
		c.paths = append(c.paths, path...)
	}
}

func WithFlags(flags map[string]*pflag.Flag) Option {
	return func(s *config) {
		s.flags = flags
	}
}

func WithEnvironmentVariables(ev map[string][]string) Option {
	return func(s *config) {
		s.environmentVariables = ev
	}
}

// WithValues sets values to be injected into the config, taking precedence over all other options.
func WithValues(values map[string]any) Option {
	return func(c *config) {
		c.values = values
	}
}

// New returns a configuration with the provided options applied. If no options are provided, the returned config
// contains only the default values.
func New(opts ...Option) (*config, error) {
	base := viper.New()
	base.SetEnvPrefix(environmentVariablePrefix)
	base.SetTypeByDefaultValue(inferConfigTypes)
	base.AutomaticEnv()
	base.SetEnvKeyReplacer(environmentVariableReplace)

	c := &config{
		base:       base,
		defaultCfg: configenv.Production,
		paths:      make([]string, 0),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.setDefault(c.defaultCfg)

	// merge the config files in the order they were passed.
	for _, path := range c.paths {
		if err := c.Merge(path); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("the specified configuration file %q doesn't exist", path)
			}
			return nil, fmt.Errorf("opening config file %q: %w", path, err)
		}
	}

	for name, values := range c.environmentVariables {
		if err := c.base.BindEnv(append([]string{name}, values...)...); err != nil {
			return nil, fmt.Errorf("binding environment variable %q to config: %w", name, err)
		}
	}

	for name, flag := range c.flags {
		if err := c.base.BindPFlag(name, flag); err != nil {
			return nil, fmt.Errorf("binding flag %q to config: %w", name, err)
		}
	}

	// merge the passed values last as they take highest precedence
	for name, value := range c.values {
		c.base.Set(name, value)
	}

	return c, nil
}

// Load reads in the configuration file specified by `path` overriding any previously set configuration with the values
// from the read config file.
// Load returns an error if the file cannot be read.
func (c *config) Load(path string) error {
	c.base.SetConfigFile(path)
	if err := c.base.ReadInConfig(); err != nil {
		return err
	}
	return nil
}

func (c *config) Merge(path string) error {
	c.base.SetConfigFile(path)
	if err := c.base.MergeInConfig(); err != nil {
		return err
	}
	return nil
}

// Write persists the current configuration to `path`.
// Write returns an error if:
//   - the path cannot be accessed
//   - the current configuration cannot be marshaled.
func (c *config) Write(path string) error {
	return c.base.WriteConfigAs(path)
}

// Current returns the current configuration.
// Current returns an error if the configuration cannot be unmarshalled.
func (c *config) Current() (types.BacalhauConfig, error) {
	out := new(types.BacalhauConfig)
	if err := c.base.Unmarshal(&out, DecoderHook); err != nil {
		return types.BacalhauConfig{}, err
	}
	return *out, nil
}

// Set sets the configuration value.
// This value won't be persisted in the config file.
// Will be used instead of values obtained via flags, config file, ENV, default.
func (c *config) Set(key string, value interface{}) {
	c.base.Set(key, value)
}

func (c *config) SetIfAbsent(key string, value interface{}) {
	if !c.base.IsSet(key) || reflect.ValueOf(c.base.Get(key)).IsZero() {
		c.Set(key, value)
	}
}

// setDefault sets the default value for the configuration.
// Default only used when no value is provided by the user via an explicit call to Set, flag, config file or ENV.
func (c *config) setDefault(config types.BacalhauConfig) {
	types.SetDefaults(config, types.WithViper(c.base))
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
