package repo

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

type RepoVersion struct {
	Version int
}

const (
	// repo versioning
	RepoVersion1    = 1
	RepoVersionFile = "repo.version"

	// user key files
	Libp2pPrivateKeyFileName = "libp2p_private_key"
	UserPrivateKeyFileName   = "user_id.pem"

	// compute paths
	ComputeStoragesPath = "executor_storages"
	PluginsPath         = "plugins"
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

func (fsr *FsRepo) writeVersion() error {
	repoVersion := RepoVersion{Version: RepoVersion1}
	versionJson, err := json.Marshal(repoVersion)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(fsr.path, RepoVersionFile), versionJson, util.OS_USER_RW)
}

func (fsr *FsRepo) readVersion() (int, error) {
	versionBytes, err := os.ReadFile(filepath.Join(fsr.path, RepoVersionFile))
	if err != nil {
		return -1, err
	}
	var version RepoVersion
	if err := json.Unmarshal(versionBytes, &version); err != nil {
		return -1, err
	}
	return version.Version, nil
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
	// check if the path is present
	if _, err := os.Stat(fsr.path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	// check if the repo version file is present
	versionPath := filepath.Join(fsr.path, RepoVersionFile)
	if _, err := os.Stat(versionPath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	version, err := fsr.readVersion()
	if err != nil {
		return false, err
	}
	if version != RepoVersion1 {
		return false, fmt.Errorf("unknown repo version %d", version)
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

	if cfg.User.KeyPath == "" {
		// if the user has not specified the location of their user key via a config file, use the default value
		cfg.User.KeyPath = filepath.Join(fsr.path, UserPrivateKeyFileName)
		config.SetUserKey(cfg.User.KeyPath)
	}

	if cfg.User.Libp2pKeyPath == "" {
		// if the user has not specified the location of their libp2p key via a config file, use the default value
		cfg.User.Libp2pKeyPath = filepath.Join(fsr.path, Libp2pPrivateKeyFileName)
		config.SetLibp2pKey(cfg.User.Libp2pKeyPath)
	}

	if cfg.Node.ExecutorPluginPath == "" {
		cfg.Node.ExecutorPluginPath = filepath.Join(fsr.path, PluginsPath)
		config.SetExecutorPluginPath(cfg.Node.ExecutorPluginPath)
	}

	if cfg.Node.ComputeStoragePath == "" {
		cfg.Node.ComputeStoragePath = filepath.Join(fsr.path, ComputeStoragesPath)
		config.SetComputeStoragesPath(cfg.Node.ComputeStoragePath)
	}

	// Using a slice of paths to minimize repetitive checks
	pathsToCheck := []string{
		cfg.User.KeyPath,
		cfg.User.Libp2pKeyPath,
		cfg.Node.ExecutorPluginPath,
		cfg.Node.ComputeStoragePath,
	}

	for _, path := range pathsToCheck {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("filepath '%s' does not exists", path)
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

func (fsr *FsRepo) Init() error {
	defaultConfig := config.ForEnvironment()
	if exists, err := fsr.Exists(); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("cannot init repo: repo already exists")
	}

	log.Info().Msgf("Initializing repo at '%s' for environment '%s'", fsr.path, config.GetConfigEnvironment())

	// 0755: Owner can read, write, execute. Others can read and execute.
	if err := os.MkdirAll(fsr.path, repoPermission); err != nil && !os.IsExist(err) {
		return err
	}

	var err error
	defaultConfig.User.KeyPath, err = fsr.ensureUserIDKey(UserPrivateKeyFileName)
	if err != nil {
		return fmt.Errorf("failed to create user key: %w", err)
	}
	defaultConfig.User.Libp2pKeyPath, err = fsr.ensureLibp2pKey(Libp2pPrivateKeyFileName)
	if err != nil {
		return fmt.Errorf("failed to create libp2p key: %w", err)
	}
	defaultConfig.Node.ExecutorPluginPath, err = fsr.ensureDir(PluginsPath)
	if err != nil {
		return fmt.Errorf("failed to create plugin dir: %w", err)
	}
	defaultConfig.Node.ComputeStoragePath, err = fsr.ensureDir(ComputeStoragesPath)
	if err != nil {
		return fmt.Errorf("failed to create executor storage dir: %w", err)
	}

	_, err = config.Init(defaultConfig, fsr.path, configName, configType)
	if err != nil {
		return err
	}
	return fsr.writeVersion()
}

func (fsr *FsRepo) ensureDir(name string) (string, error) {
	path := filepath.Join(fsr.path, name)
	if err := os.MkdirAll(path, util.OS_USER_RWX); err != nil {
		return "", fmt.Errorf("failed to create directory at at '%s': %w", path, err)
	}
	return path, nil
}

const (
	bitsPerKey = 2048 // number of bits in generated RSA keypairs
)

// ensureUserIDKey ensures that a default user ID key exists in the config dir.
func (fsr *FsRepo) ensureUserIDKey(name string) (string, error) {
	keyFile := filepath.Join(fsr.path, name)
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

func (fsr *FsRepo) ensureLibp2pKey(name string) (string, error) {

	// We include the port in the filename so that in devstack multiple nodes
	// running on the same host get different identities
	privKeyPath := filepath.Join(fsr.path, name)

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
		return privKeyPath, err
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
