package repo

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

type FsRepo struct {
	path string
}

func NewFS(path string) (*FsRepo, error) {
	expandedPath, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		path: expandedPath,
	}, nil
}

func (fsr *FsRepo) Path() (string, error) {
	if exists, err := fsr.Exists(); err != nil {
		return "", err
	} else if !exists {
		return "", fmt.Errorf("repo is uninitialized")
	}
	return fsr.path, nil
}

func (fsr *FsRepo) Exists() (bool, error) {
	if _, err := os.Stat(fsr.path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (fsr *FsRepo) Open() error {
	if exists, err := fsr.Exists(); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("repo does not exist")
	}

	cfg, err := config.Load(fsr.path, configName, configType)
	if err != nil {
		return err
	}

	// Using a slice of paths to minimize repetitive checks
	pathsToCheck := []string{
		cfg.User.UserKeyPath,
		cfg.User.Libp2pKeyPath,
		cfg.Node.ExecutorPluginPath,
		cfg.Node.ComputeStoragePath,
	}

	for _, path := range pathsToCheck {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("filepath '%s' does not exits", path)
			}
			return fmt.Errorf("failed to stat '%s': %w", path, err)
		}
	}

	return nil
}

const (
	configType     = "yaml"
	configName     = "config"
	repoPermission = 0755
)

func (fsr *FsRepo) Init(defaultConfig *types.BacalhauConfig) error {
	if exists, err := fsr.Exists(); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("cannot init repo: repo already exists")
	}

	log.Info().Msgf("Initializing repo at '%s'", fsr.path)

	// 0755: Owner can read, write, execute. Others can read and execute.
	if err := os.MkdirAll(fsr.path, repoPermission); err != nil && !os.IsExist(err) {
		return err
	}

	// Setting default configurations
	defaultConfig.Metrics.EventTracerPath = filepath.Join(fsr.path, "bacalhau-event-tracer.json")
	defaultConfig.Metrics.Libp2pTracerPath = filepath.Join(fsr.path, "bacalhau-libp2p-tracer.json")

	pathsToEnsure := []func(string) (string, error){
		ensureUserIDKey,
		ensureLibp2pKey,
		ensurePluginPath,
		ensureStoragePath,
	}

	fieldsToUpdate := []*string{
		&defaultConfig.User.UserKeyPath,
		&defaultConfig.User.Libp2pKeyPath,
		&defaultConfig.Node.ExecutorPluginPath,
		&defaultConfig.Node.ComputeStoragePath,
	}

	for i, ensureFunc := range pathsToEnsure {
		path, err := ensureFunc(fsr.path)
		if err != nil {
			return err
		}
		*fieldsToUpdate[i] = path
	}

	_, err := config.Init(defaultConfig, fsr.path, configName, configType)
	return err
}

const pluginsPath = "plugins"

func ensurePluginPath(configDir string) (string, error) {
	path := filepath.Join(configDir, pluginsPath)
	if err := os.MkdirAll(filepath.Join(configDir, pluginsPath), util.OS_USER_RWX); err != nil {
		return "", fmt.Errorf("failed to create plugin at '%s': %w", path, err)
	}
	return path, nil
}

const storagesPath = "executor_storages"

func ensureStoragePath(configDif string) (string, error) {
	path := filepath.Join(configDif, storagesPath)
	if err := os.MkdirAll(filepath.Join(configDif, storagesPath), util.OS_USER_RWX); err != nil {
		return "", fmt.Errorf("failed to create storage path at '%s': %w", path, err)
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
	keyName := "libp2p_private_key"

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

const defaultRunInfoFilename = "bacalhau.run"
const runInfoFilePermissions = 0755

func (fsr *FsRepo) WriteRunInfo(ctx context.Context, summaryShellVariablesString string) (string, error) {
	runInfoPath := filepath.Join(fsr.path, defaultRunInfoFilename)

	// TODO kill this
	devStackRunInfoPath := os.Getenv("DEVSTACK_ENV_FILE")
	if devStackRunInfoPath != "" {
		runInfoPath = devStackRunInfoPath
	}

	// Use os.Create to truncate the file if it already exists
	f, err := os.Create(runInfoPath)
	if err != nil {
		return "", err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("Failed to close run info file %s", runInfoPath)
		}
	}()

	// Set permissions to constant for read read/write only by user
	err = f.Chmod(runInfoFilePermissions)
	if err != nil {
		return "", err
	}

	_, err = f.Write([]byte(summaryShellVariablesString))
	if err != nil {
		return "", err
	}

	return runInfoPath, nil
	// TODO previous behaviour put it in these places, we may consider creating a symlink later
	/*
		if writeable, _ := filefs.IsWritable("/run"); writeable {
			writePath = "/run" // Linux
		} else if writeable, _ := filefs.IsWritable("/var/run"); writeable {
			writePath = "/var/run" // Older Linux
		} else if writeable, _ := filefs.IsWritable("/private/var/run"); writeable {
			writePath = "/private/var/run" // MacOS
		} else {
			// otherwise write to the user's dir, which should be available on all systems
			userDir, err := os.UserHomeDir()
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("Could not write to /run, /var/run, or /private/var/run, and could not get user's home dir")
				return nil
			}
			log.Warn().Msgf("Could not write to /run, /var/run, or /private/var/run, writing to %s dir instead. "+
				"This file contains sensitive information, so please ensure it is limited in visibility.", userDir)
			writePath = userDir
		}
	*/
}
