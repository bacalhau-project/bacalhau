package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true
	DefaultFileName           = "config.yaml"
)

var (
	environmentVariableReplace = strings.NewReplacer(".", "_")
	DecoderHook                = viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc())
)

type Config struct {
	// viper instance for holding user provided configuration.
	base *viper.Viper
	// the default configuration values to initialize with.
	defaultCfg interface{}

	// paths to configuration files merged from [0] to [N]
	// e.g. file at index 1 overrides index 0, index 2 overrides index 1 and 0, etc.
	paths []string

	flags map[string][]*pflag.Flag

	environmentVariables map[string][]string

	// values to inject into the config, taking highest precedence.
	values map[string]any
}

type Option = func(s *Config)

// WithDefault sets the default config to be used when no values are provided.
func WithDefault(cfg interface{}) Option {
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

func WithFlags(flags map[string][]*pflag.Flag) Option {
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
		base:                 base,
		defaultCfg:           types.Default,
		paths:                make([]string, 0),
		values:               make(map[string]any),
		environmentVariables: make(map[string][]string),
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

	for name, flags := range c.flags {
		for _, flag := range flags {
			// only if the flag has been set do we want to bind to it, this allows multiple flags
			// to bind to the same config key.
			if flag.Changed {
				switch name {
				case "ipfs.connect.deprecated":
					// allow the deprecated --ipfs-connect flag to bind to related fields in the config.
					for _, key := range []string{
						// config keys we wish to bind --ipfs-connect flag to.
						types.ResultDownloadersTypesIPFSEndpointKey,
						types.InputSourcesTypesIPFSEndpointKey,
						types.PublishersTypesIPFSEndpointKey,
					} {
						if err := c.base.BindPFlag(key, flag); err != nil {
							return nil, fmt.Errorf("binding flag %q to config: %w", name, err)
						}
					}
				case "node.type.deprecated":
					// continuing to support the deprecated --node-type flag
					// iff config values were not provided set them accordingly
					orchestrator, compute, err := getNodeType(flag.Value.String())
					if err != nil {
						return nil, err
					}
					if orchestrator {
						if _, ok := c.values[types.OrchestratorEnabledKey]; !ok {
							c.values[types.OrchestratorEnabledKey] = true
						}
					}
					if compute {
						if _, ok := c.values[types.ComputeEnabledKey]; !ok {
							c.values[types.ComputeEnabledKey] = true
						}
					}
				case "default.publisher.deprecated":
					// allow the deprecated --default-publisher flag to bind to related fields in the config.
					for _, key := range []string{
						// config keys we wish to bind --default-publisher flag to.
						types.JobDefaultsBatchTaskPublisherConfigTypeKey,
						types.JobDefaultsOpsTaskPublisherConfigTypeKey,
					} {
						if err := c.base.BindPFlag(key, flag); err != nil {
							return nil, fmt.Errorf("binding flag %q to config: %w", name, err)
						}
					}
				default:
					if err := c.base.BindPFlag(name, flag); err != nil {
						return nil, fmt.Errorf("binding flag %q to config: %w", name, err)
					}
				}
			}
		}
	}

	// merge the passed values last as they take highest precedence
	for name, value := range c.values {
		c.base.Set(name, value)
	}

	return c, nil
}

func getNodeType(input string) (requester, compute bool, err error) {
	requester = false
	compute = false
	err = nil

	// Split the string by commas, lowercase it, and clean up any extra spaces
	tokens := strings.Split(input, ",")
	for i, token := range tokens {
		tokens[i] = strings.ToLower(strings.TrimSpace(token))
	}

	for _, nodeType := range tokens {
		if nodeType == "compute" {
			compute = true
		} else if nodeType == "requester" || nodeType == "orchestrator" {
			requester = true
		} else {
			err = fmt.Errorf("invalid node type %s. Only compute and requester values are supported", nodeType)
		}
	}
	return
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

func (c *Config) Get(key string) any {
	return c.base.Get(key)
}

func (c *Config) ConfigFileUsed() string {
	return c.base.ConfigFileUsed()
}

// Unmarshal returns the current configuration.
// Unmarshal returns an error if the configuration cannot be unmarshalled.
func (c *Config) Unmarshal(out interface{}) error {
	if err := c.base.Unmarshal(&out, DecoderHook); err != nil {
		return err
	}
	return nil
}

// KeyAsEnvVar returns the environment variable corresponding to a config key
func KeyAsEnvVar(key string) string {
	return strings.ToUpper(
		fmt.Sprintf("%s_%s", environmentVariablePrefix, environmentVariableReplace.Replace(key)),
	)
}

func GenerateNodeID(ctx context.Context, nodeNameProviderType string) (string, error) {
	nodeNameProviders := map[string]idgen.NodeNameProvider{
		"hostname": idgen.HostnameProvider{},
		"aws":      idgen.NewAWSNodeNameProvider(),
		"gcp":      idgen.NewGCPNodeNameProvider(),
		"uuid":     idgen.UUIDNodeNameProvider{},
		"puuid":    idgen.PUUIDNodeNameProvider{},
	}
	nodeNameProvider, ok := nodeNameProviders[nodeNameProviderType]
	if !ok {
		return "", fmt.Errorf(
			"unknown node name provider: %s. Supported providers are: %s", nodeNameProviderType, lo.Keys(nodeNameProviders))
	}

	nodeName, err := nodeNameProvider.GenerateNodeName(ctx)
	if err != nil {
		return "", err
	}

	return nodeName, nil
}
