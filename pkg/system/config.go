package system

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

func InitConfig() error {
	configDir := os.Getenv("BACALHAU_DIR")
	//If FIL_WALLET_ADDRESS is set, assumes that ROOT_DIR is the config dir for Station
	//and not a generic environment variable set by the user
	if _, set := os.LookupEnv("FIL_WALLET_ADDRESS"); configDir == "" && set {
		configDir = os.Getenv("ROOT_DIR")
	}
	log.Debug().Msg("BACALHAU_DIR not set, using default of ~/.bacalhau")

	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home dir: %w", err)
		}
		configDir = filepath.Join(home, ".bacalhau")
	}
	fsRepo, err := repo.NewFS(configDir)
	if err != nil {
		return fmt.Errorf("failed to create repo: %w", err)
	}
	if err := fsRepo.Init(); err != nil {
		return fmt.Errorf("failed to initalize repo: %w", err)
	}
	return nil
}

const (
	sigHash = crypto.SHA256 // hash function to use for sign/verify
)

// SignForClient signs a message with the user's private ID key.
// NOTE: must be called after InitConfig() or system will panic.
func SignForClient(msg []byte) (string, error) {
	privKey, err := config_v2.GetClientPrivateKey()
	if err != nil {
		return "", err
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, sigHash, hashBytes)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyForClient verifies a signed message with the user's public ID key.
// NOTE: must be called after InitConfig() or system will panic.
func VerifyForClient(msg []byte, sig string) (bool, error) {
	pubKey, err := config_v2.GetClientPublicKey()
	if err != nil {
		return false, err
	}

	hash := sigHash.New()
	hash.Write(msg)
	hashBytes := hash.Sum(nil)

	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// A successful verification is indicated by a nil return:
	return rsa.VerifyPKCS1v15(pubKey, sigHash, hashBytes, sigBytes) == nil, nil
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
	clientID, err := config_v2.GetClientID()
	if err != nil {
		panic(fmt.Sprintf("failed to load clientID: %s", err))
	}

	return clientID
}

// GetClientPublicKey returns a base64-encoding of the user's public ID key:
// NOTE: must be called after InitConfig() or system will panic.
func GetClientPublicKey() string {
	pkstr, err := config_v2.GetClientPublicKeyString()
	if err != nil {
		panic(fmt.Sprintf("failed to load client public key: %s", err))
	}
	return pkstr
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
func EnsureConfigDir() (string, error) {
	configDir := os.Getenv("BACALHAU_DIR")
	//If FIL_WALLET_ADDRESS is set, assumes that ROOT_DIR is the config dir for Station
	//and not a generic environment variable set by the user
	if _, set := os.LookupEnv("FIL_WALLET_ADDRESS"); configDir == "" && set {
		configDir = os.Getenv("ROOT_DIR")
	}
	if configDir == "" {
		log.Debug().Msg("BACALHAU_DIR not set, using default of ~/.bacalhau")

		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "failed to get user home dir")
		}

		configDir = filepath.Join(home, ".bacalhau")
		if err = os.MkdirAll(configDir, util.OS_USER_RWX); err != nil {
			return "", errors.Wrap(err, "failed to create config dir")
		}
	} else {
		if fileinf, err := os.Stat(configDir); err != nil {
			return "", errors.Wrapf(err, "could not create or access config dir %q", configDir)
		} else if !fileinf.IsDir() {
			return "", fmt.Errorf("%q is not a directory", configDir)
		}
	}

	return configDir, nil
}

// convertToClientID converts a public key to a client ID:
func convertToClientID(key *rsa.PublicKey) string {
	hash := sigHash.New()
	hash.Write(key.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

// decodePublicKey decodes a public key from a string:
func decodePublicKey(key string) (*rsa.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return x509.ParsePKCS1PublicKey(keyBytes)
}

// InitConfigForTesting creates a fresh config setup in a temporary directory
// for testing config-related stuff and user ID message signing.
func InitConfigForTesting(t testing.TB) {
	/*
		if _, ok := os.LookupEnv("__InitConfigForTestingHasAlreadyBeenRunSoCanBeSkipped__"); ok {
			return
		}
		t.Setenv("__InitConfigForTestingHasAlreadyBeenRunSoCanBeSkipped__", "set")
	*/

	viper.Reset()
	// TODO pass a testing config.
	if err := config_v2.SetViperDefaults(config_v2.Default); err != nil {
		t.Errorf("unable to set default configuration values: %w", err)
		t.FailNow()
	}
	repoDir := t.TempDir()
	t.Setenv("BACALHAU_REPO", repoDir)
	fsRepo, err := repo.NewFS(filepath.Join(repoDir, fmt.Sprintf("bacalhau_test-%s", t.Name())))
	if err != nil {
		t.Errorf("Unable to set up config in dir %s: %w", repoDir, err)
		t.FailNow()
	}
	if err := fsRepo.Init(); err != nil {
		t.Errorf("Unable to initialize config dir %s: %w", repoDir, err)
	}
}
