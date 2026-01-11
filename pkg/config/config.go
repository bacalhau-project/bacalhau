package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	environmentVariablePrefix = "BACALHAU"
	inferConfigTypes          = true
	DefaultFileName           = "config.yaml"

	errComponent = "config"
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
//
//nolint:funlen,gocyclo // TODO: This function is very long and complex. Need to improve it.
func New(opts ...Option) (*Config, error) {
	base := viper.New()
	base.SetEnvPrefix(environmentVariablePrefix)
	base.SetTypeByDefaultValue(inferConfigTypes)
	base.AutomaticEnv()
	base.SetEnvKeyReplacer(environmentVariableReplace)

	c := &Config{
		base:                 base,
		defaultCfg:           Default,
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

	// To absolute paths for better logging. This is best effort and will not return an error if it fails.
	for i, path := range c.paths {
		c.paths[i] = AbsPathSilent(path)
	}

	// merge the config files in the order they were passed.
	for _, path := range c.paths {
		if err := c.merge(path); err != nil {
			return nil, err
		}
	}

	for name, values := range c.environmentVariables {
		if err := c.base.BindEnv(append([]string{name}, values...)...); err != nil {
			return nil, fmt.Errorf("binding environment variable %q to config: %w", name, err)
		}
	}

	if err = checkFlagConfigConflicts(c.flags, c.values); err != nil {
		return nil, err
	}

	for name, flags := range c.flags {
		for _, flag := range flags {
			// only if the flag has been set do we want to bind to it, this allows multiple flags
			// to bind to the same config key.
			if flag == nil {
				log.Error().Msgf("flag %q is nil", name)
				continue
			}
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
				case types.DataDirKey:
					// Handle relative paths for data-dir flag
					path := AbsPathSilent(flag.Value.String())
					c.base.Set(types.DataDirKey, path)
				case "default.publisher.deprecated":
					// allow the deprecated --default-publisher flag to bind to related fields in the config.
					for _, key := range []string{
						// config keys we wish to bind --default-publisher flag to.
						types.JobDefaultsBatchTaskPublisherTypeKey,
						types.JobDefaultsOpsTaskPublisherTypeKey,
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
	// allow the user to set datadir as relative paths and resolve them to absolute paths
	for name, value := range c.values {
		if name == types.DataDirKey {
			if val, ok := value.(string); ok {
				value = AbsPathSilent(val)
			}
		}
		c.base.Set(name, value)
	}

	// allow the users to set datadir to a path like ~/.bacalhau or ~/something/idk/whatever
	// and expand the path for them
	if expandedPath, err := homedir.Expand(c.base.GetString(types.DataDirKey)); err == nil {
		c.base.Set(types.DataDirKey, expandedPath)
	}

	if err := ValidatePath(c.base.GetString(types.DataDirKey)); err != nil {
		return nil, err
	}

	// if no config file was provided, we look for a config.yaml under the resolved data directory,
	// and if it exists, we create and return a new config with the resolved path.
	// we attempt this last to ensure the data-dir is resolved correctly from all config sources.
	if len(c.paths) == 0 {
		configFile := filepath.Join(c.base.GetString(types.DataDirKey), DefaultFileName)
		if _, err := os.Stat(configFile); err == nil {
			opts = append(opts, WithPaths(configFile))
			return New(opts...)
		}
	}

	log.Debug().Msgf("Config loaded from: %s, and with data-dir %s", c.paths, c.base.Get(types.DataDirKey))
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
		switch nodeType {
		case "compute":
			compute = true
		case "requester", "orchestrator":
			requester = true
		default:
			err = fmt.Errorf("invalid node type %s. Only compute and requester values are supported", nodeType)
		}
	}
	return
}

// Load reads in the configuration file specified by `path` overriding any previously set configuration with the values
// from the read config file.
// Load returns an error if the file cannot be read.
func (c *Config) Load(path string) error {
	c.base.SetConfigFile(path)
	if err := c.base.ReadInConfig(); err != nil {
		return err
	}
	return nil
}

// merge merges a new configuration file specified by `path` with the existing config.
// merge returns an error if the file cannot be read
func (c *Config) merge(path string) error {
	c.base.SetConfigFile(path)
	if err := c.base.MergeInConfig(); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("the specified configuration file %q doesn't exist", path)
		}
		return fmt.Errorf("opening config file %q: %w", path, err)
	}
	return nil
}

func (c *Config) Get(key string) any {
	return c.base.Get(key)
}

func (c *Config) ConfigFileUsed() string {
	return c.base.ConfigFileUsed()
}

// Paths returns the paths to the configuration files merged
// from lower index to higher index
func (c *Config) Paths() []string {
	return c.paths
}

// Unmarshal returns the current configuration.
// Unmarshal returns an error if the configuration cannot be unmarshalled.
func (c *Config) Unmarshal(out interface{}) error {
	if err := c.base.Unmarshal(&out, DecoderHook); err != nil {
		return err
	}
	if v, ok := out.(models.Validatable); ok {
		return v.Validate()
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

func AbsPathSilent(path string) string {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		log.Debug().Msgf("failed to expand path %s: %v", path, err)
		return path
	}
	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		log.Debug().Msgf("failed to resolve absolute path for %s: %v", path, err)
		return path
	}
	return absPath
}

func ValidatePath(path string) error {
	if path == "" {
		return bacerrors.New("data dir path is empty").
			WithHint("Provide a valid path for the data directory").
			WithComponent(errComponent).
			WithCode(bacerrors.ValidationError)
	}
	if strings.Contains(path, "$") {
		return bacerrors.Newf("data dir path %q contains a '$' character", path).
			WithHint("Note that environment variables are not expanded will be used as-is").
			WithComponent(errComponent).
			WithCode(bacerrors.ValidationError)
	}

	if !filepath.IsAbs(path) {
		return bacerrors.Newf("data dir path %q is not an absolute path", path).
			WithHint("Use an absolute path for the data directory").
			WithComponent(errComponent).
			WithCode(bacerrors.ValidationError)
	}

	return nil
}

// checkFlagConfigConflicts checks for conflicts between cli flags and config values.
// e.g. bacalhau serve --config=api.host=0.0.0.0 --api-host=0.0.0.0 should be rejected.
func checkFlagConfigConflicts(flags map[string][]*pflag.Flag, cfgValues map[string]any) error {
	for name, flagList := range flags {
		if cfgValue, exists := cfgValues[name]; exists {
			for _, flag := range flagList {
				if flag.Changed {
					return bacerrors.Newf("flag: --%s and config flag key %q cannot both be provided. Only one may be used", flag.Name,
						name).
						WithHint("Remove --%s or --config/-c %s=%v from the command", flag.Name, name, cfgValue)
				}
			}
		}
	}
	return nil
}
