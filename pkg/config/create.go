package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p/core/crypto"
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

const environmentVariablePrefix = "BACALHAU"

var environmentVariableReplace = strings.NewReplacer(".", "_")

func InitConfig(configPath string) error {
	// configure viper.
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.SetEnvPrefix(environmentVariablePrefix)
	viper.AddConfigPath(configPath)
	viper.SetEnvKeyReplacer(environmentVariableReplace)

	userKeyPath, err := ensureUserIDKey(configPath)
	if err != nil {
		return err
	}

	libp2pKeyPath, err := ensureLibp2pKey(configPath)
	if err != nil {
		return err
	}

	pluginPath, err := ensurePluginPath(configPath)
	if err != nil {
		return err
	}

	storagePath, err := ensureStoragePath(configPath)
	if err != nil {
		return err
	}

	// TODO bit of a shoe horn.
	viper.SetDefault(NodeUserUserKeyPath, userKeyPath)
	viper.SetDefault(NodeUserLibp2pKeyPath, libp2pKeyPath)
	viper.SetDefault(NodeExecutorPluginPath, pluginPath)
	viper.SetDefault(NodeComputeStoragePath, storagePath)
	viper.SetDefault(NodeMetricsEventTracerPath, filepath.Join(configPath, "bacalhau-event-tracer.json"))
	viper.SetDefault(NodeMetricsLibp2pTracerPath, filepath.Join(configPath, "bacalhau-libp2p-tracer.json"))

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
	viper.Reset()
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.SetEnvPrefix(environmentVariablePrefix)
	viper.AddConfigPath(configPath)
	viper.SetEnvKeyReplacer(environmentVariableReplace)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("viper failed to read config: %w", err)
	}

	userKeyPath, err := ensureUserIDKey(configPath)
	if err != nil {
		return err
	}

	libp2pKeyPath, err := ensureLibp2pKey(configPath)
	if err != nil {
		return err
	}

	pluginPath, err := ensurePluginPath(configPath)
	if err != nil {
		return err
	}

	storagePath, err := ensureStoragePath(configPath)
	if err != nil {
		return err
	}

	// TODO bit of a shoe horn.
	viper.SetDefault(NodeUserUserKeyPath, userKeyPath)
	viper.SetDefault(NodeUserLibp2pKeyPath, libp2pKeyPath)
	viper.SetDefault(NodeExecutorPluginPath, pluginPath)
	viper.SetDefault(NodeComputeStoragePath, storagePath)
	viper.SetDefault(NodeMetricsEventTracerPath, filepath.Join(configPath, "bacalhau-event-tracer.json"))
	viper.SetDefault(NodeMetricsLibp2pTracerPath, filepath.Join(configPath, "bacalhau-libp2p-tracer.json"))

	// now register env vars
	viper.AutomaticEnv()
	// TODO maybe not the right place, this also has it's own configuration.
	telemetry.SetupFromEnvs()
	return nil
}

const pluginsPath = "plugins"

func ensurePluginPath(configDir string) (string, error) {
	path := filepath.Join(configDir, pluginsPath)
	if err := os.MkdirAll(filepath.Join(configDir, pluginsPath), util.OS_USER_RWX); err != nil {
		return "", err
	}
	return path, nil
}

const storagesPath = "executor_storages"

func ensureStoragePath(configDif string) (string, error) {
	path := filepath.Join(configDif, storagesPath)
	if err := os.MkdirAll(filepath.Join(configDif, storagesPath), util.OS_USER_RWX); err != nil {
		return "", err
	}
	return path, nil
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

func ensureLibp2pKey(configDir string) (string, error) {
	keyName := fmt.Sprintf("private_key.%d", viper.GetInt(NodeLibp2pSwarmPort))

	// We include the port in the filename so that in devstack multiple nodes
	// running on the same host get different identities
	privKeyPath := filepath.Join(configDir, keyName)

	if _, err := os.Stat(privKeyPath); errors.Is(err, os.ErrNotExist) {
		// Private key does not exist - create and write it

		// Creates a new RSA key pair for this host.
		prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, bitsPerKey, rand.Reader)
		if err != nil {
			log.Error().Err(err)
			return "", err
		}

		keyOut, err := os.OpenFile(privKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.OS_USER_RW)
		if err != nil {
			return "", fmt.Errorf("failed to open key.pem for writing: %v", err)
		}
		privBytes, err := crypto.MarshalPrivateKey(prvKey)
		if err != nil {
			return "", fmt.Errorf("unable to marshal private key: %v", err)
		}
		// base64 encode privBytes
		b64 := base64.StdEncoding.EncodeToString(privBytes)
		_, err = keyOut.WriteString(b64 + "\n")
		if err != nil {
			return "", fmt.Errorf("failed to write to key file: %v", err)
		}
		if err := keyOut.Close(); err != nil {
			return "", fmt.Errorf("error closing key file: %v", err)
		}
		log.Debug().Msgf("wrote %s", privKeyPath)
	} else {
		return "", err
	}

	// Now that we've ensured the private key is written to disk, read it! This
	// ensures that loading it works even in the case where we've just created
	// it.

	// read the private key
	keyBytes, err := os.ReadFile(privKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %v", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %v", err)
	}
	// parse the private key
	_, err = crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	return privKeyPath, nil
}
