package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const ENV_PREFIX = "BACALHAU"

const DEFAULT_LOG_FILE = "/tmp/bacalhau.log"
const DEFAULT_IPFS_DOCKER_IMAGE = "binocarlos/bacalhau-ipfs-sidebar-image:v1"
const DEFAULT_SWARM_ANNOUNCE_IP = "127.0.0.1"

var GlobalConfig *viper.Viper

// the path to the bacalhau config file
// if provided - this should be a path to a yaml file
// otherwise we will use $HOME/.bacalhau/config.yaml
// all values can be override with BACALHAU_XXX env variables
func getConfigFilePath() (string, error) {
	configFilePath := os.Getenv("BACALHAU_CONFIG_FILE")
	if configFilePath == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configFileDir := fmt.Sprintf("%s/.bacalhau", homedir)
		err = os.MkdirAll(configFileDir, 0700)
		if err != nil {
			return "", err
		}
		configFilePath = fmt.Sprintf("%s/config.yaml", configFileDir)
		file, err := os.Create(configFilePath)
		if err != nil {
			return "", err
		}
		file.Close()
	} else {
		if _, err := os.Stat(configFilePath); err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	return configFilePath, nil
}

func CreateConfig(cmd *cobra.Command) (*viper.Viper, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}
	config := viper.New()
	// any env variable prefixed with BACALHAU_ will be used as a config
	config.SetEnvPrefix(ENV_PREFIX)
	config.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	config.AutomaticEnv()
	if cmd != nil {
		err = config.BindPFlags(cmd.Flags())
		if err != nil {
			return nil, err
		}
	}

	config.SetDefault("logfile", DEFAULT_LOG_FILE)
	config.SetDefault("ipfs_docker_sidecar_image", DEFAULT_IPFS_DOCKER_IMAGE)
	config.SetDefault("swarm_announce_ip", DEFAULT_SWARM_ANNOUNCE_IP)

	// do we have a specific config file to read or are we seafching the system for one?
	configFilePath := os.Getenv("BACALHAU_CONFIG_FILE")
	if configFilePath != "" {
		if _, err := os.Stat(configFilePath); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		file, err := os.Open(configFilePath)
		defer func() {
			err = file.Close()
			if err != nil {
				log.Error().Msg(err.Error())
			}
		}()
		if err != nil {
			return nil, err
		}
		err = config.ReadConfig(file)
		if err != nil {
			return nil, err
		}
	} else {
		config.SetConfigName("config") // name of config file (without extension)
		config.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
		config.AddConfigPath("/etc/bacalhau")
		config.AddConfigPath("$HOME/.bacalhau")
		config.AddConfigPath(".")
		config.AddConfigPath(configPath)

		if err := config.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found; ignore error if desired
			} else {
				return nil, err
			}
		}
	}
	GlobalConfig = config
	return config, nil
}

func GetConfig() (*viper.Viper, error) {
	if GlobalConfig == nil {
		return nil, fmt.Errorf("no config loaded")
	}
	return GlobalConfig, nil
}
