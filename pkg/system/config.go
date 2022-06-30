package system

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const chmodUserAll = 0700 // read, write, execute permissions for user

// InitConfig ensures that a bacalhau config file exists and loads it.
func InitConfig() error {
	configDir, err := ensureConfigDir()
	if err != nil {
		return fmt.Errorf("failed to init config dir: %w", err)
	}

	configFile, err := ensureConfigFile(configDir)
	if err != nil {
		return fmt.Errorf("failed to init config file: %w", err)
	}

	viper.SetConfigFile(configFile) // provided or created config file
	viper.SetConfigType("yaml")     // config is always a yaml file
	viper.AutomaticEnv()            // try to read config from env if possible
	viper.SetEnvPrefix("bacalhau")  // BACALHAU_<key> is encoding for env vars
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	return nil
}

// ensureDefaultConfigDir ensures that a bacalhau config dir exists.
func ensureConfigDir() (string, error) {
	configDir := os.Getenv("BACALHAU_DIR")
	if configDir == "" {
		log.Debug().Msg("BACALHAU_DIR not set, using default of ~/.bacalhau")

		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home dir: %w", err)
		}

		configDir = fmt.Sprintf("%s/.bacalhau", home)
		if err = os.MkdirAll(configDir, chmodUserAll); err != nil {
			return "", fmt.Errorf("failed to create config dir: %w", err)
		}
	} else {
		if _, err := os.Stat(configDir); err != nil {
			return "", fmt.Errorf("failed to stat config dir '%s': %w",
				configDir, err)
		}
	}

	return configDir, nil
}

// ensureConfigFile ensures that BACALHAU_DIR/config.yaml exists.
func ensureConfigFile(configDir string) (string, error) {
	configFile := fmt.Sprintf("%s/config.yaml", configDir)
	if _, err := os.Stat(configFile); err != nil {
		if os.IsNotExist(err) {
			log.Debug().Msgf(
				"config file %s does not exist, creating default", configFile)

			var file *os.File
			file, err = os.Create(configFile)
			if err != nil {
				return "", fmt.Errorf("failed to create config file: %w", err)
			}
			if err = file.Close(); err != nil {
				return "", fmt.Errorf("failed to close config file: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to stat config file: %w", err)
		}
	}

	return configFile, nil
}
