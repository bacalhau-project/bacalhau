package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config/migrations"
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
	configType     = "yaml"
	configName     = "config"
	configFileMode = 0666
)

func Init(path string) (types.BacalhauConfig, error) {
	// derive the default config for the specified environment.
	defaultConfig := ForEnvironment()

	// set default values for path dependent config.
	defaultConfig.User.KeyPath = filepath.Join(path, UserPrivateKeyFileName)
	defaultConfig.User.Libp2pKeyPath = filepath.Join(path, Libp2pPrivateKeyFileName)
	defaultConfig.Node.ExecutorPluginPath = filepath.Join(path, PluginsPath)
	defaultConfig.Node.ComputeStoragePath = filepath.Join(path, ComputeStoragesPath)
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
	defaultConfig.Update.CheckStatePath = filepath.Join(path, UpdateCheckStatePath)

	return initConfig(path, WithDefaultConfig(defaultConfig), WithFileHandler(ReadConfigHandler))
}

func Migrate(path string) error {
	// check if the config file exists, if one is not found we don't need to migrate it
	configPath := filepath.Join(path, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to stat config file for migration: %w", err)
	}

	// open the config file
	f, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file at %q: %w", configPath, err)
	}

	// read it all and unmarshal into yaml
	b, err := io.ReadAll(f)
	if err != nil {
		if err := f.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close config file after failing to read it.")
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}
	// we can close the file now that we read everything.
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close config file: %w", err)
	}

	var cfg types.BacalhauConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	// get all the migrations we need to apply to it.
	migs, err := migrations.GetMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migration list: %w", err)
	}

	// apply the migrations
	currentCfg := cfg
	for _, m := range migs {
		log.Info().Msgf("applying migration sequence %d", m.Sequence())
		currentCfg, err = m.Migrate(currentCfg)
		if err != nil {
			return err
		}
	}
	log.Info().Msgf("config migration complete")

	// marshal the migrated config back to yaml
	marshaledCfg, err := yaml.Marshal(currentCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal migrated config: %w", err)
	}

	// open the file for writing and truncate it.
	fw, err := os.OpenFile(configPath, os.O_WRONLY|os.O_TRUNC, configFileMode)
	if err != nil {
		return fmt.Errorf("failed to open config file for writing at %q: %w", configPath, err)
	}
	defer fw.Close()

	// write the marshaled data back to the file
	if _, err := fw.Write(marshaledCfg); err != nil {
		return fmt.Errorf("failed to write migrated config to file: %w", err)
	}

	return nil
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
