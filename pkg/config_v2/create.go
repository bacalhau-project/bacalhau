package config_v2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

func SetViperDefaults(config BacalhauConfig) error {
	setDefaults(config)
	return nil
}

const configType = "toml"
const configName = "config"

func InitConfig(configPath string) error {
	// configure viper.
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configPath)
	viper.SetEnvPrefix("BACALHAU")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	userKeyPath, err := ensureUserIDKey(configPath)
	if err != nil {
		return err
	}

	// TODO bit of a shoe horn.
	viper.SetDefault(NodeUserUserKeyPath, userKeyPath)

	// now write the default values to it.
	if err := viper.SafeWriteConfig(); err != nil {
		return fmt.Errorf("viper failed to write config: %w", err)
	}

	// now register env vars
	viper.AutomaticEnv()
	// TODO maybe not the right place, this also has it's own configuration.
	telemetry.SetupFromEnvs()
	return nil
}

func LoadConfig(configPath string) error {
	// configure viper.
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configPath)
	viper.SetEnvPrefix("BACALHAU")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("viper failed to read config: %w", err)
	}

	userKeyPath, err := ensureUserIDKey(configPath)
	if err != nil {
		return err
	}

	// TODO bit of a shoe horn.
	viper.SetDefault(NodeUserUserKeyPath, userKeyPath)

	// now register env vars
	viper.AutomaticEnv()
	// TODO maybe not the right place, this also has it's own configuration.
	telemetry.SetupFromEnvs()
	return nil
}

const (
	bitsPerKey = 2048 // number of bits in generated RSA keypairs
)

// ensureUserIDKey ensures that a default user ID key exists in the config dir.
func ensureUserIDKey(configDir string) (string, error) {
	keyFile := fmt.Sprintf("%s/user_id.pem", configDir)
	if _, err := os.Stat(keyFile); err != nil {
		if os.IsNotExist(err) {
			log.Debug().Msgf(
				"user ID key file '%s' does not exist, creating one", keyFile)

			var key *rsa.PrivateKey
			key, err = rsa.GenerateKey(rand.Reader, bitsPerKey)
			if err != nil {
				return "", fmt.Errorf("failed to generate private key: %w", err)
			}

			keyBytes := x509.MarshalPKCS1PrivateKey(key)
			keyBlock := pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: keyBytes,
			}

			var file *os.File
			file, err = os.Create(keyFile)
			if err != nil {
				return "", fmt.Errorf("failed to create key file: %w", err)
			}
			if err = pem.Encode(file, &keyBlock); err != nil {
				return "", fmt.Errorf("failed to encode key file: %w", err)
			}
			if err = file.Close(); err != nil {
				return "", fmt.Errorf("failed to close key file: %w", err)
			}
			if err = os.Chmod(keyFile, util.OS_USER_RW); err != nil {
				return "", fmt.Errorf("failed to set permission on key file: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to stat user ID key '%s': %w",
				keyFile, err)
		}
	}

	return keyFile, nil
}
