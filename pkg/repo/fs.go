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

	// UpdateCheckStatePath is the update check paths.
	UpdateCheckStatePath = "update.json"
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

// join joins path elements with fsr.path
func (fsr *FsRepo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.path}, paths...)...)
}

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

	// check if a config file is already present, even though the repo is uninitialized
	// users may still place a config file in a repo (we do this for our terraform deployments)
	// we should attempt to load the config file if it's present.
	if _, err := os.Stat(fsr.join(config.FileName)); err == nil {
		if err := c.Load(fsr.join(config.FileName)); err != nil {
			return fmt.Errorf("failed to load config file present in repo: %w", err)
		}
	}

	fsr.EnsureRepoPathsConfigured(c)

	cfg, err := c.Current()
	if err != nil {
		return err
	}

	if err := initRepoFiles(cfg); err != nil {
		return fmt.Errorf("failed to initialize repo: %w", err)
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()
	return fsr.writeVersion(RepoVersion3)
}

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

	// load the configuration for the repo.
	// Repos without a config file are still valid. So check if one is present.
	if _, err := os.Stat(fsr.join(config.FileName)); err == nil {
		if err := c.Load(fsr.join(config.FileName)); err != nil {
			return fmt.Errorf("failed to load config file present in repo: %w", err)
		}
	}

	// modifies the config to include keys for accessing repo paths if they are not set.
	// This ensures either user provided paths are valid to default paths for the repo are set.
	fsr.EnsureRepoPathsConfigured(c)

	cfg, err := c.Current()
	if err != nil {
		return err
	}

	// ensure the loaded config has valid fields as they pertain to the filesystem
	// e.g. user key and libp2p files exists, storage paths exist, etc.
	if err := validateRepoConfig(cfg); err != nil {
		return fmt.Errorf("failed to validate repo config: %w", err)
	}

	// derive an installationID from the client ID loaded from the repo.
	if cfg.User.InstallationID == "" {
		ID, _ := config.GetClientID(cfg.User.KeyPath)
		uuidFromUserID := uuid.NewSHA1(uuid.New(), []byte(ID))
		c.Set(types.UserInstallationID, uuidFromUserID.String())
	}

	// TODO we should be initializing the file as a part of creating the repo, instead of sometime later.
	if cfg.Update.CheckStatePath == "" {
		c.Set(types.UpdateCheckStatePath, fsr.join(UpdateCheckStatePath))
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()

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

// modifies the config to include keys for accessing repo paths
func (fsr *FsRepo) EnsureRepoPathsConfigured(c config.ReadWriter) {
	c.SetIfAbsent(types.AuthTokensPath, fsr.join(config.TokensPath))
	c.SetIfAbsent(types.UserKeyPath, fsr.join(config.UserPrivateKeyFileName))
	c.SetIfAbsent(types.NodeExecutorPluginPath, fsr.join(config.PluginsPath))

	// NB(forrest): pay attention to the subtle name difference here
	c.SetIfAbsent(types.NodeComputeStoragePath, fsr.join(config.ComputeStoragesPath))

	c.SetIfAbsent(types.UpdateCheckStatePath, fsr.join(config.UpdateCheckStatePath))
	c.SetIfAbsent(types.NodeClientAPITLSAutoCertCachePath, fsr.join(config.AutoCertCachePath))
	c.SetIfAbsent(types.UserLibp2pKeyPath, fsr.join(config.Libp2pPrivateKeyFileName))
	c.SetIfAbsent(types.NodeNetworkStoreDir, fsr.join(config.OrchestratorStorePath, config.NetworkTransportStore))

	c.SetIfAbsent(types.NodeRequesterJobStorePath, fsr.join(config.OrchestratorStorePath, "jobs.db"))
	c.SetIfAbsent(types.NodeComputeExecutionStorePath, fsr.join(config.ComputeStorePath, "executions.db"))
}
