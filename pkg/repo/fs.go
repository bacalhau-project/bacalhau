package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
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
func (fsr *FsRepo) Init(cfg types2.Bacalhau) error {
	if exists, err := fsr.Exists(); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("cannot init repo: repo already exists")
	}

	log.Info().Msgf("Initializing repo at '%s' for environment '%s'", fsr.path, config.GetConfigEnvironment())

	// if it takes longer than 5 seconds to get the node name from a provider, fail
	nameCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	nodeName, err := getNodeID(nameCtx, cfg.NameProvider)
	if err != nil {
		return err
	}

	// 0755: Owner can read, write, execute. Others can read and execute.
	if err := os.MkdirAll(fsr.path, repoPermission); err != nil && !os.IsExist(err) {
		return err
	}

	// TODO this should be a part of the config.
	telemetry.SetupFromEnvs()

	if err := fsr.WriteVersion(Version4); err != nil {
		return fmt.Errorf("failed to persist repo version: %w", err)
	}

	if err := fsr.WriteNodeName(nodeName); err != nil {
		return fmt.Errorf("failed to persist node name: %w", err)
	}
	return nil
}

// Open opens an existing repo, returning an error if the repo is uninitialized.
func (fsr *FsRepo) Open() error {
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

// EnsureRepoPathsConfigured modifies the config to include keys for accessing repo paths
func (fsr *FsRepo) EnsureRepoPathsConfigured(c config.ReadWriter) {
	c.SetIfAbsent(types.AuthTokensPath, fsr.join(config.TokensPath))
	c.SetIfAbsent(types.UserKeyPath, fsr.join(config.UserPrivateKeyFileName))
	c.SetIfAbsent(types.NodeExecutorPluginPath, fsr.join(config.PluginsPath))

	// NB(forrest): pay attention to the subtle name difference here
	c.SetIfAbsent(types.NodeComputeStoragePath, fsr.join(config.ComputeStoragesPath))

	c.SetIfAbsent(types.NodeClientAPITLSAutoCertCachePath, fsr.join(config.AutoCertCachePath))
	c.SetIfAbsent(types.NodeNetworkStoreDir, fsr.join(config.OrchestratorStorePath, config.NetworkTransportStore))

	c.SetIfAbsent(types.NodeRequesterJobStorePath, fsr.join(config.OrchestratorStorePath, "jobs.db"))
	c.SetIfAbsent(types.NodeComputeExecutionStorePath, fsr.join(config.ComputeStorePath, "executions.db"))
}

// join joins path elements with fsr.path
func (fsr *FsRepo) join(paths ...string) string {
	return filepath.Join(append([]string{fsr.path}, paths...)...)
}

func getNodeID(ctx context.Context, nodeNameProviderType string) (string, error) {
	nodeNameProviders := map[string]idgen.NodeNameProvider{
		"hostname": idgen.HostnameProvider{},
		"aws":      idgen.NewAWSNodeNameProvider(),
		"gcp":      idgen.NewGCPNodeNameProvider(),
		"uuid":     idgen.UUIDNodeNameProvider{},
		"puuid":    idgen.PUUIDNodeNameProvider{},
	}
	nodeNameProvider, ok := nodeNameProviders[nodeNameProviderType]
	if !ok {
		return "", fmt.Errorf(
			"unknown node name provider: %s. Supported providers are: %s", nodeNameProviderType, lo.Keys(nodeNameProviders))
	}

	nodeName, err := nodeNameProvider.GenerateNodeName(ctx)
	if err != nil {
		return "", err
	}

	return nodeName, nil
}
