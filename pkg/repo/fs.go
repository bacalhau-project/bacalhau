package repo

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configfx"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

const (
	repoPermission         = 0755
	defaultRunInfoFilename = "bacalhau.run"
	runInfoFilePermissions = 0755

	// UpdateCheckStatePath is the update check paths.
	UpdateCheckStatePath = "update.json"
)

type Repo interface {
	Init(v ...*viper.Viper) error
	Open(v ...*viper.Viper) error
	Config() (*configfx.Config, error)
	WriteRunInfo(ctx context.Context, summary string) (string, error)
	Path() (string, error)
	Exists() (bool, error)
	Version() (int, error)
}

var _ Repo = (*FsRepo)(nil)

type FsRepoParams struct {
	Path       string
	Migrations *MigrationManager
}

type FsRepo struct {
	path       string
	Migrations *MigrationManager
	config     *configfx.Config
}

func NewFS(params FsRepoParams) (*FsRepo, error) {
	expandedPath, err := homedir.Expand(params.Path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		path:       expandedPath,
		Migrations: params.Migrations,
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
	if !IsValidVersion(version) {
		return false, NewUnknownRepoVersionError(version)
	}
	return true, nil
}

// Version returns the version of the repo.
func (fsr *FsRepo) Version() (int, error) {
	return fsr.readVersion()
}

func (fsr *FsRepo) Init(v ...*viper.Viper) error {
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

	// TODO consider passing the config in to repo init
	c := configfx.New()
	if len(v) > 1 {
		panic("too many vipers")
	}
	if len(v) == 1 {
		c = configfx.NewWithViper(v[0])
	}
	cfg, err := c.Init(fsr.path)
	if err != nil {
		return err
	}

	if err := initRepoFiles(cfg); err != nil {
		return fmt.Errorf("failed to initialize repo: %w", err)
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()
	fsr.config = c
	return fsr.writeVersion(RepoVersion3)
}

func (fsr *FsRepo) Open(v ...*viper.Viper) error {
	// if the repo does not exist we cannot open it.
	if exists, err := fsr.Exists(); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("repo does not exist")
	}

	if fsr.Migrations != nil {
		if err := fsr.Migrations.Migrate(*fsr); err != nil {
			return fmt.Errorf("failed to migrate repo: %w", err)
		}
	}

	// TODO consider passing the config in to repo init
	c := configfx.New()
	if len(v) > 1 {
		panic("too many vipers")
	}
	if len(v) == 1 {
		c = configfx.NewWithViper(v[0])
	}
	// load the configuration for the repo.
	cfg, err := c.Load(fsr.path)
	if err != nil {
		return err
	}
	fsr.config = c

	// ensure the loaded config has valid fields as they pertain to the filesystem
	// e.g. user key and libp2p files exists, storage paths exist, etc.
	if err := validateRepoConfig(cfg); err != nil {
		return fmt.Errorf("failed to validate repo config: %w", err)
	}

	// derive an installationID from the client ID loaded from the repo.
	if cfg.User.InstallationID == "" {
		ID, _ := fsr.GetClientID()
		uuidFromUserID := uuid.NewSHA1(uuid.New(), []byte(ID))
		c.SetValue(types.UserInstallationID, uuidFromUserID.String())
	}

	// TODO we should be initializing the file as a part of creating the repo, instead of sometime later.
	if cfg.Update.CheckStatePath == "" {
		cfg.Update.CheckStatePath = filepath.Join(fsr.path, UpdateCheckStatePath)
		c.SetValue(types.UpdateCheckStatePath, cfg.Update.CheckStatePath)
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()

	return nil
}

func (fsr *FsRepo) Config() (*configfx.Config, error) {
	if fsr.config == nil {
		return nil, fmt.Errorf("repo doesn't have config developer error")
	}
	return fsr.config, nil
}

// WritePersistedConfigs will write certain values from the resolved config to the persisted config.
// These include fields for configurations that must not change between version updates, such as the
// execution store and job store paths, in case we change their default values in future updates.
func (fsr *FsRepo) WritePersistedConfigs() error {
	// a viper config instance that is only based on the config file.
	name := fsr.config.Viper().Get(types.NodeName)
	_ = name
	resolvedCfg, err := fsr.config.Current()
	if err != nil {
		return err
	}
	configFilePath := filepath.Join(fsr.path, configfx.ConfigFileName)
	viperWriter := viper.New()
	viperWriter.SetTypeByDefaultValue(true)
	viperWriter.SetConfigFile(configFilePath)

	// read existing config if it exists.
	if err := viperWriter.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	var fileCfg types.BacalhauConfig
	if err := viperWriter.Unmarshal(&fileCfg, configfx.DecoderHook); err != nil {
		return err
	}

	// check if any of the values that we want to write are not set in the config file.
	var doWrite bool
	var logMessage strings.Builder
	set := func(key string, value interface{}) {
		viperWriter.Set(key, value)
		logMessage.WriteString(fmt.Sprintf("\n%s:\t%v", key, value))
		doWrite = true
	}
	emptyStoreConfig := types.JobStoreConfig{}
	if fileCfg.Node.Compute.ExecutionStore == emptyStoreConfig {
		set(types.NodeComputeExecutionStore, resolvedCfg.Node.Compute.ExecutionStore)
	}
	if fileCfg.Node.Requester.JobStore == emptyStoreConfig {
		set(types.NodeRequesterJobStore, resolvedCfg.Node.Requester.JobStore)
	}
	if fileCfg.Node.Name == "" && resolvedCfg.Node.Name != "" {
		set(types.NodeName, resolvedCfg.Node.Name)
	}
	if doWrite {
		log.Info().Msgf("Writing to config file %s:%s", configFilePath, logMessage.String())
		return viperWriter.WriteConfig()
	}
	return nil
}

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
	// TODO previous behavior put it in these places, we may consider creating a symlink later
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

// loadClientID loads a hash identifying a user based on their ID key.
func (fsr *FsRepo) GetClientID() (string, error) {
	key, err := fsr.loadUserIDKey()
	if err != nil {
		return "", fmt.Errorf("failed to load user ID key: %w", err)
	}

	return convertToClientID(&key.PublicKey), nil
}

const (
	sigHash = crypto.SHA256 // hash function to use for sign/verify
)

// convertToClientID converts a public key to a client ID:
func convertToClientID(key *rsa.PublicKey) string {
	hash := sigHash.New()
	hash.Write(key.N.Bytes())
	hashBytes := hash.Sum(nil)

	return fmt.Sprintf("%x", hashBytes)
}

func (fsr *FsRepo) GetClientPublicKey() (*rsa.PublicKey, error) {
	privKey, err := fsr.loadUserIDKey()
	if err != nil {
		return nil, err
	}
	return &privKey.PublicKey, nil
}

func (fsr *FsRepo) GetClientPrivateKey() (*rsa.PrivateKey, error) {
	privKey, err := fsr.loadUserIDKey()
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

// loadUserIDKey loads the user ID key from whatever source is configured.
func (fsr *FsRepo) loadUserIDKey() (*rsa.PrivateKey, error) {
	keyFile, found := fsr.config.GetString(types.UserKeyPath)
	if !found {
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
		return nil, fmt.Errorf("failed to decode user ID key file %q", keyFile)
	}

	// TODO: #3159 Add support for both rsa _and_ ecdsa private keys, see crypto.PrivateKey.
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
