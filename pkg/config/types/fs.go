package types

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

func ensureDir(path string) error {
	exists, err := dirExists(path)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return mkDir(path)
}

func mkDir(path string) error {
	// dir cannot already exist
	exists, err := dirExists(path)
	if err != nil {
		return fmt.Errorf("checking if directory exists at path %q: %w", path, err)
	}
	if exists {
		return os.ErrExist
	}

	// create the directory and return the full path
	if err := os.MkdirAll(path, util.OS_USER_RWX); err != nil {
		return fmt.Errorf("failed to create directory at at %q: %w", path, err)
	}
	return nil
}

func dirExists(path string) (bool, error) {
	// stat path to check if exists
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// it exists, but we failed to stat it, return err
		return false, fmt.Errorf("failed to check if directory exists at path %q: %w", path, err)
	}
	// if the path exists, ensure it's a directory
	if !stat.IsDir() {
		return false, fmt.Errorf("path %q is a file, expected directory", path)
	}
	return true, nil
}

const (
	// bitsPerKey number of bits in generated RSA keypairs for the user key.
	bitsPerKey = 2048 // number of bits in generated RSA keypairs
)

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
	file, err = os.Create(path) //nolint:gosec // G304: path from config system, application controlled
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if err = pem.Encode(file, &keyBlock); err != nil {
		return fmt.Errorf("failed to encode key file: %w", err)
	}

	if err = os.Chmod(path, util.OS_USER_RW); err != nil {
		return fmt.Errorf("failed to set permission on key file: %w", err)
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
