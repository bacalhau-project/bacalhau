package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inlocalstore"
	memcomputestore "github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	memjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
)

// cribbed from lotus

type FsRepo struct {
	path string
}

func NewFS(path string) (*FsRepo, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return nil, err
	}

	return &FsRepo{
		path: path,
	}, nil

}

func (fsr *FsRepo) Path() (string, error) {
	exists, err := fsr.Exists()
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("repo is uninitalized")
	}
	return fsr.path, nil
}

func (fsr *FsRepo) Exists() (bool, error) {
	_, err := os.Stat(fsr.path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// InitExecutionStore must be called after Init
func (fsr *FsRepo) InitExecutionStore(ctx context.Context, prefix string) (store.ExecutionStore, error) {
	if exists, err := fsr.Exists(); err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("repo is uninitialized, cannot create ExecutionStore")
	}
	stateRootDir := filepath.Join(fsr.path, fmt.Sprintf("%s-execution-state", prefix))
	if err := os.MkdirAll(stateRootDir, os.ModePerm); err != nil {
		return nil, err
	}
	// load the compute nodes execution store config
	var storeCfg config_v2.StorageConfig
	if err := config_v2.GetConfigForKey(config_v2.NodeComputeExecutionStore, &storeCfg); err != nil {
		return nil, err
	}
	var (
		store store.ExecutionStore
		err   error
	)
	switch storeCfg.Type {
	case config_v2.BoltDB:
		path := storeCfg.Path
		if path == "" {
			path = filepath.Join(stateRootDir, fmt.Sprintf("%s-compute.db", prefix))
		}
		store, err = boltdb.NewStore(ctx, path)
		if err != nil {
			return nil, err
		}
	case config_v2.InMemory:
		store = memcomputestore.NewStore()
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
	return inlocalstore.NewPersistentExecutionStore(inlocalstore.PersistentJobStoreParams{
		Store:   store,
		RootDir: stateRootDir,
	})
}

// InitJobStore must be called after Init
func (fsr *FsRepo) InitJobStore(prefix string) (jobstore.Store, error) {
	if exists, err := fsr.Exists(); err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("repo is uninitialized, cannot create JobStore")
	}
	// load the compute nodes execution store config
	var storeCfg config_v2.StorageConfig
	if err := config_v2.GetConfigForKey(config_v2.NodeRequesterJobStore, &storeCfg); err != nil {
		return nil, err
	}
	switch storeCfg.Type {
	case config_v2.BoltDB:
		path := storeCfg.Path
		if path == "" {
			path = filepath.Join(fsr.path, fmt.Sprintf("%s-requester.db", prefix))
		}
		return boltjobstore.NewBoltJobStore(path)
	case config_v2.InMemory:
		return memjobstore.NewInMemoryJobStore(), nil
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
}

func (fsr *FsRepo) Init() error {
	exist, err := fsr.Exists()
	if err != nil {
		return err
	}
	if exist {
		log.Debug().Msgf("Repo found at '%s", fsr.path)
		return config_v2.LoadConfig(fsr.path)
	}

	log.Info().Msgf("Initializing repo at '%s'", fsr.path)
	// 0755 The owner can read, write, and execute, while others can read and execute.
	err = os.MkdirAll(fsr.path, 0755) //nolint: gosec
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := config_v2.InitConfig(fsr.path); err != nil {
		return err
	}

	return nil
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
