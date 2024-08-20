package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

// V3Migration updates the repo, replacing repo.version and update.json with system_metadata.yaml.
// It does the following:
// - Creates system_metadata.yaml with repo version 4.
// - Sets the last update check time in system_metadata.yaml to Unix time zero.
// - If an installationID is present in the config, its value is persisted to system_metadata.yaml.
// - Removes update.json if the file is present.
// - Removes repo.version.
// - Creates a new directory .bacalhau/orchestrator.
// - Moves contents of .bacalhau/orchestrator_store to .bacalhau/orchestrator and renames jobs.db to state_boltdb.db.
// - Removes .bacalhau/orchestrator_store.
// - Creates a new directory .bacalhau/compute.
// - Moves executions.db from .bacalhau/compute_store to .bacalhau/compute/state_boltdb.db.
// - Creates a new directory .bacalhau/compute/executions.
// - Moves contents of .bacalhau/execution_store to .bacalhau/compute/executions.
// - Removes ./bacalhau/execution_store.
// - If a user has configured a custom user key path, the configured value is copied to .bacalhau/user_id.pem.
// - If a user has configured a custom auth tokens path, the configured value is copied to .bacalhau/tokens.json.
var V3Migration = StagedMigration(
	repo.Version3,
	repo.Version4,
	func(r repo.FsRepo) error {
		repoPath, err := r.Path()
		if err != nil {
			return err
		}
		_, fileCfg, err := readConfig(r)
		if err != nil {
			return err
		}
		// migrate from the repo.version file to the system_metadata.yaml file.
		{
			// Initialize the SystemMetadataFile in the staging directory
			if err := r.WriteVersion(repo.Version4); err != nil {
				return err
			}
			if err := r.WriteLastUpdateCheck(time.UnixMilli(0)); err != nil {
				return err
			}
			if fileCfg.User.InstallationID != "" {
				if err := r.WriteInstallationID(fileCfg.User.InstallationID); err != nil {
					return err
				}
			}

			// ignore this error as the file may not exist
			_ = os.Remove(filepath.Join(repoPath, "update.json"))
			// remove the legacy repo version file
			if err := os.Remove(filepath.Join(repoPath, repo.LegacyVersionFile)); err != nil {
				return fmt.Errorf("removing legacy repo version file: %w", err)
			}
		}

		// migrate to the new repo structure
		{
			// if the user provided a non-standard path we will move it to the migrated repo
			// if the user didn't provide a path, no copy required as the location of the file in the repo
			// is unchanged.
			if fileCfg.User.KeyPath != "" {
				if err := copyFile(fileCfg.User.KeyPath, filepath.Join(repoPath, types.UserKeyFileName)); err != nil {
					return fmt.Errorf("copying user key file: %w", err)
				}
			}

			// if the user provided a non-standard path we will move it to the migrated repo
			// if the user didn't provide a path, no copy required as the location of the file in the repo
			// is unchanged.
			if fileCfg.Auth.TokensPath != "" {
				if err := copyFile(fileCfg.Auth.TokensPath, filepath.Join(repoPath, types.AuthTokensFileName)); err != nil {
					return fmt.Errorf("copying auth tokens file: %w", err)
				}
			}

			// create the new compute store directory
			computeDirPath := filepath.Join(repoPath, types.ComputeDirName)
			if err := os.Mkdir(computeDirPath, util.OS_USER_RWX); err != nil {
				return fmt.Errorf("creating compute dir path: %w", err)
			}

			// if the user has configured a path, migration from it
			if fileCfg.Node.Compute.ExecutionStore.Path != "" {
				if err := copyFile(
					fileCfg.Node.Compute.ExecutionStore.Path,
					filepath.Join(repoPath, types.ComputeDirName, types.ExecutionStoreFileName),
				); err != nil {
					return fmt.Errorf("copying execution database: %w", err)
				}
			} else {
				// else use the default location
				if err := copyFile(
					filepath.Join(repoPath, "compute_store", "executions.db"),
					filepath.Join(repoPath, types.ComputeDirName, types.ExecutionStoreFileName)); err != nil {
					return fmt.Errorf("copying execution database: %w", err)
				}
			}

			// remove the old compute_store
			if err := os.RemoveAll(filepath.Join(repoPath, "compute_store")); err != nil {
				return fmt.Errorf("removing %s: %w", filepath.Join(repoPath, "compute_store"), err)
			}

			// create the new orchestrator store directory
			orchestratorDirPath := filepath.Join(repoPath, types.OrchestratorDirName)
			if err := os.Mkdir(orchestratorDirPath, util.OS_USER_RWX); err != nil {
				return fmt.Errorf("creating orchestrator dir path: %w", err)
			}

			// if the user has configured a path, migration from it
			if fileCfg.Node.Requester.JobStore.Path != "" {
				if err := copyFile(
					fileCfg.Node.Requester.JobStore.Path,
					filepath.Join(repoPath, types.OrchestratorDirName, types.JobStoreFileName),
				); err != nil {
					return fmt.Errorf("copying job database: %w", err)
				}
			} else {
				// else use the default location
				if err := copyFile(
					filepath.Join(repoPath, "orchestrator_store", "jobs.db"),
					filepath.Join(repoPath, types.OrchestratorDirName, types.JobStoreFileName)); err != nil {
					return fmt.Errorf("copying execution database: %w", err)
				}
			}

			// remove the old orchestrator_store
			if err := os.RemoveAll(filepath.Join(repoPath, "orchestrator_store")); err != nil {
				return fmt.Errorf("removing %s: %w", filepath.Join(repoPath, "orchestrator_store"), err)
			}

			from := fileCfg.Node.ComputeStoragePath
			if from == "" {
				from = filepath.Join(repoPath, "executor_storages")
			}
			to := filepath.Join(repoPath, types.ComputeDirName, types.ExecutionDirName)
			log.Info().Str("from", from).Str("to", to).Msg("copying executor storages")
			fromExists, fromErr := fileExists(from)
			toExists, toErr := fileExists(to)
			event := log.Info().Bool("fromExists", fromExists).Bool("toExists", toExists)
			if fromErr != nil {
				event.Str("fromErr", fromErr.Error())
			}
			if toErr != nil {
				event.Str("toErr", toErr.Error())
			}
			event.Msg("BUILD KIT DEBUG")
			if err := os.Rename(from, to); err != nil {
				return fmt.Errorf("migrating executor storages: %w", err)
			}
		}
		return nil
	},
)

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
