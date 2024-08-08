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
	// viper instance for holding user provided configuration
	base *viper.Viper
	// the default configuration values to initialize with
	defaultCfg types.BacalhauConfig

	loadXDG  string
	loadRepo string
}

type Option = func(s *config)

func WithDefaultConfig(cfg types.BacalhauConfig) Option {
	return func(c *config) {
		c.defaultCfg = cfg
	}
}

func WithXDGPath(path string) Option {
	return func(c *config) {
		c.loadXDG = path
	}
}

func WithRepoPath(path string) Option {
	return func(c *config) {
		c.loadRepo = path
	}
}

func New(opts ...Option) *config {
	xdgPath, err := os.UserConfigDir()
	if err != nil {
		log.Warn().Err(err).Msg("failed to find user config dir")
	} else {
		xdgPath = filepath.Join(xdgPath, "bacalhau", FileName)
	}

	c := &config{
		base:       viper.New(),
		defaultCfg: configenv.Production,
		loadRepo:   filepath.Join(viper.GetString("repo"), FileName),
		loadXDG:    xdgPath,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.base.SetEnvPrefix(environmentVariablePrefix)
	c.base.SetTypeByDefaultValue(inferConfigTypes)
	c.base.AutomaticEnv()
	c.base.SetEnvKeyReplacer(environmentVariableReplace)

	// 1. Set default fields in the config, these are used for fields without a corresponding file/flag/envvars value.
	// e.g. if no flags or environment values are provided these defaults are used.
	c.setDefault(c.defaultCfg)

	// 2. Attempt to read a config file from the bacalhau repo, taking precedence
	// over default values.
	//
	// Note:
	// - The presence of a config file is not mandatory at this stage.
	// - Any values read from this config file will override the default values set in the previous step.
	//
	// Logging:
	// - A warning will be logged if:
	//   1. A config file was found but could not be read.
	if c.loadRepo != "" {
		if err := c.Merge(c.loadRepo); err != nil {
			if !os.IsNotExist(err) {
				log.Warn().Err(err).Msg("failed to read config from bacalhau repo")
			}
		}
	}

	// 3. Attempt to read a config file from the default user configuration directory, taking precedence over repo
	// config.
	// Location specified by: https://specifications.freedesktop.org/basedir-spec/latest/
	//
	// Note:
	// - The presence of a config file is not mandatory at this stage.
	// - Any values read from this config file will override the values set in repo config.
	//
	// Logging:
	// - A warning will be logged if:
	//   1. A config file was found but could not be read.
	//   2. `os.UserConfigDir` returns an error, which typically indicates that the $HOME environment variable is
	//      not defined (this is a rare occurrence).
	if c.loadXDG != "" {
		if err := c.Merge(c.loadXDG); err != nil {
			if !os.IsNotExist(err) {
				log.Warn().Err(err).Msg("failed to read config from user config dir")
			}
		}
	}

	// 4. Finally, merge in any configuration values present on the global viper instance, taking precedence over user
	// config directory.
	// These values come from --config flags provided by the users and take the highest precedence.
	settings := viper.GetViper().AllSettings()
	if err := c.base.MergeConfigMap(settings); err != nil {
		// NB(forrest): this method never errors: https://github.com/spf13/viper/blob/cc53fac037475edaec5cd2cae73e6c3cc5caef9e/viper.go#L1564
		// I suspect the method signature contains an error return value as it returned an error at one point and the
		// signature was left unchanged for compatibility reasons.
		panic(fmt.Sprintf("DEVELOPER ERROR: viper.MergeConfigMap returned unexpected error: %s", err))
	}

	// NB(forrest): from the above comments set 1 has lowest precedence, step 4 has highest precedence.
	return c
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
