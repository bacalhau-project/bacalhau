package system

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	chmodUserAll = 0700          // read, write, execute permissions for user
	bitsPerKey   = 2048          // number of bits in generated RSA keypairs
	sigHash      = crypto.SHA256 // hash function to use for sign/verify
)

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

	userIDKey, err := ensureUserIDKey(configDir)
	if err != nil {
		return fmt.Errorf("failed to init user ID key file: %w", err)
	}
	viper.SetDefault("user-id-key", userIDKey) // rsa key for identifying user

	// Settings and initialisation for viper:
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

// InitConfigForTesting creates a fresh config setup in a temporary directory
// for testing config-related stuff and user ID message signing.
func InitConfigForTesting(t *testing.T) {
	configDir, err := ioutil.TempDir("", "bacalhau-test")
	assert.NoError(t, err)
	assert.NoError(t, os.Setenv("BACALHAU_DIR", configDir))
	assert.NoError(t, InitConfig())
}

// SignForUser signs a message with the user's private id key.
func SignForUser(msg []byte) ([]byte, error) {
	key, err := loadUserIDKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load user ID key: %w", err)
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, key, sigHash, hashBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return sig, nil
}

// VerifyForUser verifies a signed message with the user's public id key.
func VerifyForUser(msg, sig []byte) (bool, error) {
	key, err := loadUserIDKey()
	if err != nil {
		return false, fmt.Errorf("failed to load user ID key: %w", err)
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	// A successful verification is indicated by a nil return:
	return rsa.VerifyPKCS1v15(&key.PublicKey, sigHash, hashBytes, sig) == nil, nil
}

// GetClientID returns a hash identifying a user based on their id key.
func GetClientID() (string, error) {
	key, err := loadUserIDKey()
	if err != nil {
		return "", fmt.Errorf("failed to load user ID key: %w", err)
	}

	hash := sigHash.New()
	hash.Write(key.PublicKey.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes), nil
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
			if err = os.Chmod(keyFile, chmodUserAll); err != nil {
				return "", fmt.Errorf("failed to set permission on key file: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to stat user ID key '%s': %w",
				keyFile, err)
		}
	}

	return keyFile, nil
}

// loadUserIDKey loads the user ID key from whatever source is configured.
func loadUserIDKey() (*rsa.PrivateKey, error) {
	keyFile := viper.GetString("user-id-key")
	if keyFile == "" {
		return nil, fmt.Errorf("config error: user-id-key not set")
	}

	file, err := os.Open(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open user ID key file: %w", err)
	}
	defer file.Close()

	keyBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read user ID key file: %w", err)
	}

	keyBlock, _ := pem.Decode(keyBytes)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode user ID key file")
	}

	// TODO: Add support for both rsa _and_ ecdsa private keys, see cryto.PrivateKey.
	//       Since we have access to the private key we can hack it by signing a
	//       message twice and comparing them, rather than verifying directly.
	// ecdsaKey, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse user: %w", err)
	// }

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID key file: %w", err)
	}

	return key, nil
}
