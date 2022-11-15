package system

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	bitsPerKey = 2048          // number of bits in generated RSA keypairs
	sigHash    = crypto.SHA256 // hash function to use for sign/verify
)

var (
	globalClientID  string          // global cache of client ID
	globalUserIDKey *rsa.PrivateKey // global cache of user ID key
)

// InitConfig ensures that a bacalhau config file exists and loads it.
// NOTE: this will override the global config cache if called twice.
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
	if err = viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	// Cache user ID key related data so we don't have to constantly load
	// it from disk, and so that we fail fast if something is wrong:
	globalUserIDKey, err = loadUserIDKey()
	if err != nil {
		return fmt.Errorf("failed to load user ID key: %w", err)
	}

	globalClientID, err = loadClientID()
	if err != nil {
		return fmt.Errorf("failed to load client ID: %w", err)
	}

	newTraceProvider()

	return nil
}

type testingT interface {
	TempDir() string
}

// InitConfigForTesting creates a fresh config setup in a temporary directory
// for testing config-related stuff and user ID message signing.
// NOTE: this will overwrite the global config cache if called twice.
func InitConfigForTesting(t testingT) error {
	configDir := t.TempDir()
	err := os.Setenv("BACALHAU_DIR", configDir)
	if err != nil {
		return err
	}
	err = InitConfig()
	if err != nil {
		return err
	}
	return nil
}

// SignForClient signs a message with the user's private ID key.
// NOTE: must be called after InitConfig() or system will panic.
func SignForClient(msg []byte) (string, error) {
	if globalUserIDKey == nil {
		panic("must call InitConfig() before calling SignForClient()")
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, globalUserIDKey, sigHash, hashBytes)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyForClient verifies a signed message with the user's public ID key.
// NOTE: must be called after InitConfig() or system will panic.
func VerifyForClient(msg []byte, sig string) (bool, error) {
	if globalUserIDKey == nil {
		panic("must call InitConfig() before calling VerifyForClient()")
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// A successful verification is indicated by a nil return:
	return rsa.VerifyPKCS1v15(&globalUserIDKey.PublicKey, sigHash, hashBytes, sigBytes) == nil, nil
}

// Verify verifies a signed message with the given encoding of a public key.
// Returns non nil if the key is invalid.
func Verify(msg []byte, sig, publicKey string) error {
	key, err := decodePublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// A successful verification is indicated by a nil return
	return rsa.VerifyPKCS1v15(key, sigHash, hashBytes, sigBytes)
}

// GetClientID returns a hash identifying a user based on their ID key.
// NOTE: must be called after InitConfig() or system will panic.
func GetClientID() string {
	if globalClientID == "" {
		panic("must call InitConfig() before calling GetClientID()")
	}

	return globalClientID
}

// GetClientPublicKey returns a base64-encoding of the user's public ID key:
// NOTE: must be called after InitConfig() or system will panic.
func GetClientPublicKey() string {
	if globalUserIDKey == nil {
		panic("must call InitConfig() before calling GetPublicKey()")
	}

	return encodePublicKey(&globalUserIDKey.PublicKey)
}

// PublicKeyMatchesID returns true if the given base64-encoded public key and
// the given client ID correspond to each other:
func PublicKeyMatchesID(publicKey, clientID string) (bool, error) {
	pkey, err := decodePublicKey(publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to decode public key: %w", err)
	}

	return clientID == convertToClientID(pkey), nil
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
		if err = os.MkdirAll(configDir, util.OS_USER_RWX); err != nil {
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
			if err = os.Chmod(keyFile, util.OS_USER_RWX); err != nil {
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
	defer closer.CloseWithLogOnError("user ID key file", file)

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

// loadClientID loads a hash identifying a user based on their ID key.
func loadClientID() (string, error) {
	key, err := loadUserIDKey()
	if err != nil {
		return "", fmt.Errorf("failed to load user ID key: %w", err)
	}

	return convertToClientID(&key.PublicKey), nil
}

// convertToClientID converts a public key to a client ID:
func convertToClientID(key *rsa.PublicKey) string {
	hash := sigHash.New()
	hash.Write(key.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

// encodePublicKey encodes a public key as a string:
func encodePublicKey(key *rsa.PublicKey) string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PublicKey(key))
}

// decodePublicKey decodes a public key from a string:
func decodePublicKey(key string) (*rsa.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return x509.ParsePKCS1PublicKey(keyBytes)
}
