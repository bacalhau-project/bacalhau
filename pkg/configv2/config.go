package configv2

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

const (
	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true
)

var (
	environmentVariableReplace = strings.NewReplacer(".", "_")
	DecoderHook                = viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())
)

type Config struct {
	// viper instance for holding user provided configuration.
	base *viper.Viper
	// the default configuration values to initialize with.
	defaultCfg types.Validatable

	// paths to configuration files merged from [0] to [N]
	// e.g. file at index 1 overrides index 0, index 2 overrides index 1 and 0, etc.
	paths []string

	flags map[string]*pflag.Flag

	environmentVariables map[string][]string

	// values to inject into the config, taking highest precedence.
	values map[string]any
}

type Option = func(s *Config)

// WithDefault sets the default config to be used when no values are provided.
func WithDefault(cfg types.Validatable) Option {
	return func(c *Config) {
		c.defaultCfg = cfg
	}
}

// WithPaths sets paths to configuration files to be loaded
// paths to configuration files merged from [0] to [N]
// e.g. file at index 1 overrides index 0, index 2 overrides index 1 and 0, etc.
func WithPaths(path ...string) Option {
	return func(c *Config) {
		c.paths = append(c.paths, path...)
	}
}

func WithFlags(flags map[string]*pflag.Flag) Option {
	return func(s *Config) {
		s.flags = flags
	}
}

func WithEnvironmentVariables(ev map[string][]string) Option {
	return func(s *Config) {
		s.environmentVariables = ev
	}
}

// WithValues sets values to be injected into the config, taking precedence over all other options.
func WithValues(values map[string]any) Option {
	return func(c *Config) {
		c.values = values
	}
}

// New returns a configuration with the provided options applied. If no options are provided, the returned config
// contains only the default values.
func New(opts ...Option) (*Config, error) {
	base := viper.New()
	base.SetEnvPrefix(environmentVariablePrefix)
	base.SetTypeByDefaultValue(inferConfigTypes)
	base.AutomaticEnv()
	base.SetEnvKeyReplacer(environmentVariableReplace)

	c := &Config{
		base:       base,
		defaultCfg: Default,
		paths:      make([]string, 0),
	}
	for _, opt := range opts {
		opt(c)
	}

	var defaultMap map[string]interface{}
	err := mapstructure.Decode(c.defaultCfg, &defaultMap)
	if err != nil {
		return nil, err
	}

	if err := c.base.MergeConfigMap(defaultMap); err != nil {
		return nil, err
	}

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
func (c *Config) Load(path string) error {
	log.Info().Msgf("loading config file: %q", path)
	c.base.SetConfigFile(path)
	if err := c.base.ReadInConfig(); err != nil {
		return err
	}
	return nil
}

// Merge merges a new configuration file specified by `path` with the existing config.
// Merge returns an error if the file cannot be read
func (c *Config) Merge(path string) error {
	log.Info().Msgf("merging config file: %q", path)
	c.base.SetConfigFile(path)
	if err := c.base.MergeInConfig(); err != nil {
		return err
	}
	return nil
}

// Unmarshal returns the current configuration.
// Unmarshal returns an error if the configuration cannot be unmarshalled.
func (c *Config) Unmarshal(out types.Validatable) error {
	if err := c.base.Unmarshal(&out, DecoderHook); err != nil {
		return err
	}
	if err := out.Validate(); err != nil {
		return err
	}
	return nil
}
