package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

const (
	repoPermission         = 0755
	defaultRunInfoFilename = "bacalhau.run"
	runInfoFilePermissions = 0755
)

type FsRepoParams struct {
	Path       string
	Migrations *MigrationManager
}

type FsRepo struct {
	path       string
	Migrations *MigrationManager
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

// Path returns the filesystem path to of the repo directory.
func (fsr *FsRepo) Path() (string, error) {
	if exists, err := fsr.Exists(); err != nil {
		return "", err
	} else if !exists {
		return "", fmt.Errorf("repo is uninitialized")
	}
	return fsr.path, nil
}

// Exists returns true if the repo exists and is valid, false otherwise.
func (fsr *FsRepo) Exists() (bool, error) {
	// check if the path is present
	if _, err := os.Stat(fsr.path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	version, err := fsr.Version()
	if err != nil {
		// if the repo version does not exist, then the repo is uninitialized, we don't need to error.
		if os.IsNotExist(err) {
			return false, nil
		}
		// if the repo version does exist, but could not be read this is an error.
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

// Init initializes a new repo, returning an error if the repo already exists.
func (fsr *FsRepo) Init(c config.ReadWriter) error {
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

	// create files required by repo
	if err := fsr.initializeRepoFiles(); err != nil {
		return fmt.Errorf("`initializing` repo: %w", err)
	}

	// TODO(forrest) remove this block once we delete the old config, its required for for config validation to pass
	{
		// modifies the config to include keys for accessing repo paths if they are not set.
		// This ensures either user provided paths are valid to default paths for the repo are set.
		fsr.EnsureRepoPathsConfigured(c)
		// TODO this should be a part of the config.
		telemetry.SetupFromEnvs()
	}

	return fsr.WriteVersion(Version4)
}

// Open opens an existing repo, returning an error if the repo is uninitialized.
func (fsr *FsRepo) Open(c config.ReadWriter) error {
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

	// create files required by repo
	if err := fsr.openRepoFiles(); err != nil {
		return fmt.Errorf("opening repo: %w", err)
	}

	// TODO(forrest) remove this block once we delete the old config, its required for for config validation to pass
	{
		// modifies the config to include keys for accessing repo paths if they are not set.
		// This ensures either user provided paths are valid to default paths for the repo are set.
		fsr.EnsureRepoPathsConfigured(c)
		cfg, err := c.Current()
		if err != nil {
			return err
		}

		// derive an installationID from the client ID loaded from the repo.
		if cfg.User.InstallationID == "" {
			ID, _ := config.GetClientID(cfg.User.KeyPath)
			uuidFromUserID := uuid.NewSHA1(uuid.New(), []byte(ID))
			c.Set(types.UserInstallationID, uuidFromUserID.String())
		}
		// TODO this should be a part of the config.
		telemetry.SetupFromEnvs()
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
}

func (fsr *FsRepo) UserKeyPath() (string, error) {
	return fsr.getFile(UserKeyFile)
}

func (fsr *FsRepo) AuthTokensPath() (string, error) {
	return fsr.ensureFile(AuthTokensFile)
}

func (fsr *FsRepo) OrchestratorDir() (string, error) {
	return fsr.ensureDir(OrchestratorDirKey)
}

func (fsr *FsRepo) NetworkTransportDir() (string, error) {
	return fsr.ensureDir(NetworkTransportDirKey)
}

func (fsr *FsRepo) ComputeDir() (string, error) {
	return fsr.ensureDir(ComputeDirKey)
}

func (fsr *FsRepo) ExecutionDir() (string, error) {
	return fsr.ensureDir(ExecutionDirKey)
}

func (fsr *FsRepo) EnginePluginsDir() (string, error) {
	return fsr.ensureDir(EnginePluginsDirKey)
}

// EnsureRepoPathsConfigured modifies the config to include keys for accessing repo paths
func (fsr *FsRepo) EnsureRepoPathsConfigured(c config.ReadWriter) {
	c.SetIfAbsent(types.AuthTokensPath, fsr.join(AuthTokensFile))
	c.SetIfAbsent(types.UserKeyPath, fsr.join(UserKeyFile))
	c.SetIfAbsent(types.NodeExecutorPluginPath, fsr.join(EnginePluginsDirKey))

	// NB(forrest): pay attention to the subtle name difference here
	c.SetIfAbsent(types.NodeComputeStoragePath, fsr.join(ExecutionDirKey))

	c.SetIfAbsent(types.NodeClientAPITLSAutoCertCachePath, fsr.join(AutoCertCachePath))
	c.SetIfAbsent(types.NodeNetworkStoreDir, fsr.join(NetworkTransportDirKey))

	c.SetIfAbsent(types.NodeRequesterJobStorePath, fsr.join(OrchestratorDirKey, "jobs.db"))
	c.SetIfAbsent(types.NodeComputeExecutionStorePath, fsr.join(ComputeDirKey, "executions.db"))
}

// join joins path elements with fsr.path
func (fsr *FsRepo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.path}, paths...)...)
}
