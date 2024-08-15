package repo

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

// directories
const (
	// root directories
	AutoCertCachePath = "autocert-cache"

	// orchestrator directories
	OrchestratorDirKey     = "orchestrator_store"
	NetworkTransportDirKey = OrchestratorDirKey + "/" + "nats-store"

	// compute directories
	ComputeDirKey          = "compute_store"
	ExecutionDirKey        = ComputeDirKey + "/" + "executions"
	EnginePluginsDirKey    = ComputeDirKey + "/" + "plugins" + "/" + "engines"
	TLSAutoCertCacheDirKey = ""
)

// files
const (
	UserKeyFile = "user_id.pem"
	// bitsPerKey number of bits in generated RSA keypairs for the user key.
	bitsPerKey = 2048 // number of bits in generated RSA keypairs

	AuthTokensFile = "tokens.json"
)

func (fsr *FsRepo) initializeRepoFiles() error {
	path, err := fsr.mkFile(UserKeyFile, func(f *os.File) error {
		var key *rsa.PrivateKey
		key, err := rsa.GenerateKey(rand.Reader, bitsPerKey)
		if err != nil {
			return fmt.Errorf("failed to generate private key: %w", err)
		}

		keyBytes := x509.MarshalPKCS1PrivateKey(key)
		keyBlock := pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		}
		if err := pem.Encode(f, &keyBlock); err != nil {
			return fmt.Errorf("failed to encode user key file: %w", err)
		}

		if err := f.Chmod(util.OS_USER_RW); err != nil {
			return fmt.Errorf("failed to set permission on user key file: %w", err)
		}
		return nil
	})
	log.Info().Str("path", path).Msg("initialize user key")
	return err
}

func (fsr *FsRepo) openRepoFiles() error {
	path, err := fsr.getFile(UserKeyFile)
	log.Info().Str("path", path).Msg("loaded user key")
	return err
}

func (fsr *FsRepo) ensureDir(in string) (string, error) {
	path, err := fsr.getDir(in)
	if err != nil {
		if os.IsNotExist(err) {
			return fsr.mkDir(in)
		}
		return "", err
	}
	return path, nil
}

func (fsr *FsRepo) getDir(in string) (string, error) {
	path := fsr.join(in)
	// if the repo does not exist fail.
	if exists, err := fsr.Exists(); err != nil {
		return "", fmt.Errorf("opening %s: critial error reading repo: %w", path, err)
	} else if !exists {
		return "", fmt.Errorf("opening %s: repo does not exist", path)
	}

	// if the dir does not exist fail.
	if exists, err := dirExists(path); err != nil {
		return "", err
	} else if !exists {
		return "", os.ErrNotExist
	}

	return path, nil
}

func (fsr *FsRepo) ensureFile(in string) (string, error) {
	path, err := fsr.getFile(in)
	if err != nil {
		if os.IsNotExist(err) {
			return fsr.mkFile(in, func(f *os.File) error {
				// noop, this method simply ensures the file exists,
				return nil
			})
		}
		return "", err
	}
	return path, nil
}

func (fsr *FsRepo) getFile(in string) (string, error) {
	path := fsr.join(in)
	// if the repo does not exist fail.
	if exists, err := fsr.Exists(); err != nil {
		return "", fmt.Errorf("opening %s: critial error reading repo: %w", path, err)
	} else if !exists {
		return "", fmt.Errorf("opening %s: repo does not exist", path)
	}

	// if the file does not exist fail.
	if exists, err := fileExists(path); err != nil {
		return "", err
	} else if !exists {
		return "", os.ErrNotExist
	}

	return path, nil
}

// mkDir creates a directory named `path` as a child of the repo dir.
// It returns an error if:
//  1. the repo doesn't exist
//  2. the directory at `path` already exists
func (fsr *FsRepo) mkDir(in string) (string, error) {
	path := fsr.join(in)

	// dir cannot already exist
	exists, err := dirExists(path)
	if err != nil {
		return "", fmt.Errorf("checking if directory exists at path %q: %w", path, err)
	}
	if exists {
		return "", os.ErrExist
	}

	// create the directory and return the full path
	if err := os.MkdirAll(path, util.OS_USER_RWX); err != nil {
		return "", fmt.Errorf("failed to create directory at at %q: %w", path, err)
	}
	return path, nil
}

func (fsr *FsRepo) mkFile(in string, writeFn func(f *os.File) error) (string, error) {
	path := fsr.join(in)

	// the file cannot already exist
	exists, err := fileExists(path)
	if err != nil {
		return "", fmt.Errorf("checking if file exists at path %q: %w", path, err)
	}
	if exists {
		return "", os.ErrExist
	}

	// create the file, apply the write method, return the full file path
	var file *os.File
	file, err = os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create file %q: %w", path, err)
	}
	defer file.Close()

	if err := writeFn(file); err != nil {
		return "", fmt.Errorf("writing file %q: %w", path, err)
	}

	return path, nil
}
