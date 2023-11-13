package repo

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

// initRepoFiles initializes all files required for a valid bacalhau repo.
func initRepoFiles(cfg types.BacalhauConfig) error {
	if err := initUserIDKey(cfg.User.KeyPath); err != nil {
		return fmt.Errorf("failed to create user key: %w", err)
	}

	if err := initLibp2pKey(cfg.User.Libp2pKeyPath); err != nil {
		return fmt.Errorf("failed to create libp2p key: %w", err)
	}

	if err := initDir(cfg.Node.ExecutorPluginPath); err != nil {
		return fmt.Errorf("failed to create plugin dir: %w", err)
	}

	if err := initDir(cfg.Node.ComputeStoragePath); err != nil {
		return fmt.Errorf("failed to create executor storage dir: %w", err)
	}

	if err := initDir(cfg.Node.ServerAPI.TLS.AutoCertCachePath); err != nil {
		return fmt.Errorf("failed to create tls auto certificate path: %w", err)
	}

	return nil
}

// validateRepoConfig ensures all files exist for a valid bacalhau repo.
func validateRepoConfig(cfg types.BacalhauConfig) error {
	if exists, err := fileExists(cfg.User.KeyPath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("user key file does not exist at: %q", cfg.User.KeyPath)
	}

	if exists, err := fileExists(cfg.User.Libp2pKeyPath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("libp2p key file does not exist at: %q", cfg.User.Libp2pKeyPath)
	}

	if exists, err := fileExists(cfg.Node.ExecutorPluginPath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("executor plugin path does not exist at: %q", cfg.User.Libp2pKeyPath)
	}

	if exists, err := fileExists(cfg.Node.ComputeStoragePath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("compute storage path does not exist at: %q", cfg.Node.ComputeStoragePath)
	}

	if exists, err := fileExists(cfg.Node.ServerAPI.TLS.AutoCertCachePath); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("TLS auto certification cache path does not exist at: %q", cfg.Node.ServerAPI.TLS.AutoCertCachePath)
	}

	return nil
}

func fileExists(path string) (bool, error) {
	// Check if the file exists
	_, err := os.Stat(path)
	if err == nil {
		// File exists
		return true, nil
	} else if !os.IsNotExist(err) {
		// os.Stat returned an error other than "file does not exist"
		return false, fmt.Errorf("failed to check if file exists at path: %w", err)
	}
	// file does not exist
	return false, nil
}

// initDir will create a user directory at the specified path. It returns an error if a directory is already
// present at the provided path.
func initDir(path string) error {
	// Check if the file exists, fail if it does, create it if it doesn't
	exists, err := fileExists(path)
	if err != nil {
		return fmt.Errorf("failed to check dir at path: %w", err)
	}
	if exists {
		// user key already exists.
		return fmt.Errorf("dir already exists at path: %s", path)
	}

	if err := os.MkdirAll(path, util.OS_USER_RWX); err != nil {
		return fmt.Errorf("failed to create directory at at '%s': %w", path, err)
	}
	return nil
}

const (
	// bitsPerKey number of bits in generated RSA keypairs for the libp2p and user key.
	bitsPerKey = 2048 // number of bits in generated RSA keypairs
)

// initUserIDKey will create a user key at the specified path. It returns an error if a file is already
// present at the provided path.
func initUserIDKey(path string) error {
	// Check if the file exists, fail if it does, create it if it doesn't
	exists, err := fileExists(path)
	if err != nil {
		return fmt.Errorf("failed to check user key file at path: %w", err)
	}
	if exists {
		// user key already exists.
		return fmt.Errorf("user key file already exists at path: %s", path)
	}

	// File does not exist, proceed with initialization
	log.Debug().Msgf("initializing user ID key file at '%s'", path)

	var key *rsa.PrivateKey
	key, err = rsa.GenerateKey(rand.Reader, bitsPerKey)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyBlock := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	var file *os.File
	file, err = os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer file.Close()

	if err = pem.Encode(file, &keyBlock); err != nil {
		return fmt.Errorf("failed to encode key file: %w", err)
	}

	if err = os.Chmod(path, util.OS_USER_RW); err != nil {
		return fmt.Errorf("failed to set permission on key file: %w", err)
	}

	return nil
}

// initLibp2pKey will create a libp2p key at the specified path. It returns an error if a file is already
// present at the provided path.
func initLibp2pKey(path string) error {
	// Check if the file exists, fail if it does, create it if it doesn't
	exists, err := fileExists(path)
	if err != nil {
		return fmt.Errorf("failed to check libp2p key file at path: %w", err)
	}
	if exists {
		// user key already exists.
		return fmt.Errorf("libp2p key file already exists at path: %s", path)
	}

	// File does not exist, proceed with initialization
	log.Debug().Msgf("initializing libp2p key file at '%s'", path)

	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, bitsPerKey, rand.Reader)
	if err != nil {
		log.Error().Err(err)
		return err
	}

	keyOut, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.OS_USER_RW)
	if err != nil {
		return fmt.Errorf("failed to open key.pem for writing: %v", err)
	}
	privBytes, err := crypto.MarshalPrivateKey(prvKey)
	if err != nil {
		return fmt.Errorf("unable to marshal private key: %v", err)
	}
	// base64 encode privBytes
	b64 := base64.StdEncoding.EncodeToString(privBytes)
	_, err = keyOut.WriteString(b64 + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to key file: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("error closing key file: %v", err)
	}
	log.Debug().Msgf("wrote %s", path)

	// Now that we've ensured the private key is written to disk, read it! This
	// ensures that loading it works even in the case where we've just created
	// it.

	{
		// read the private key
		keyBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read private key: %v", err)
		}
		// base64 decode keyBytes
		b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
		if err != nil {
			return fmt.Errorf("failed to decode private key: %v", err)
		}
		// parse the private key
		_, err = crypto.UnmarshalPrivateKey(b64)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %v", err)
		}
	}

	return nil
}
