package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
	legacy_types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
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
var V3Migration = repo.NewMigration(
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
				if err := copyFile(fileCfg.User.KeyPath, filepath.Join(repoPath, cfgtypes.UserKeyFileName)); err != nil {
					return fmt.Errorf("copying user key file: %w", err)
				}
			}

			// if the user provided a non-standard path we will move it to the migrated repo
			// if the user didn't provide a path, no copy required as the location of the file in the repo
			// is unchanged.
			if fileCfg.Auth.TokensPath != "" {
				if err := copyFile(fileCfg.Auth.TokensPath, filepath.Join(repoPath, cfgtypes.AuthTokensFileName)); err != nil {
					return fmt.Errorf("copying auth tokens file: %w", err)
				}
			}

			if err := migrateOrchestratorStore(repoPath, fileCfg.Node.Requester.JobStore); err != nil {
				return err
			}

			if err := migrateComputeStore(repoPath, fileCfg.Node.Compute.ExecutionStore); err != nil {
				return err
			}
		}

		// iff there is a config file in the repo, try and move it to $XDG_CONFIG_HOME/bacalhau
		{
			oldConfigFilePath := filepath.Join(repoPath, config_legacy.FileName)
			if _, err := os.Stat(oldConfigFilePath); err == nil {
				if err := r.WriteInstallationID(fileCfg.User.InstallationID); err != nil {
					return fmt.Errorf("migrating installation id: %w", err)
				}
				if err := r.WriteNodeName(fileCfg.Node.Name); err != nil {
					return fmt.Errorf("migrating node name: %w", err)
				}
				newConfigType, err := config.MigrateV1(fileCfg)
				if err != nil {
					return fmt.Errorf("migrating to new config: %w", err)
				}
				// ensure the repo path of the config points to the repo
				newConfigType.DataDir = repoPath
				userConfigDir, err := os.UserConfigDir()
				if err == nil {
					newConfigDir := filepath.Join(userConfigDir, "bacalhau")
					if err := os.Mkdir(newConfigDir, util.OS_USER_RWX); err != nil {
						return err
					}
					newConfigFilePath := filepath.Join(newConfigDir, config_legacy.FileName)
					if err := os.Rename(oldConfigFilePath, newConfigFilePath); err != nil {
						return err
					}
					newConfigBytes, err := yaml.Marshal(&newConfigType)
					if err != nil {
						return err
					}
					newConfigFile, err := os.OpenFile(newConfigFilePath, os.O_RDWR|os.O_TRUNC, util.OS_USER_RWX)
					if err != nil {
						return err
					}
					defer newConfigFile.Close()
					if _, err := newConfigFile.Write(newConfigBytes); err != nil {
						return err
					}
				}
			}

		}
		return nil
	},
)

func migrateComputeStore(repoPath string, config legacy_types.JobStoreConfig) error {
	oldComputeDir := filepath.Join(repoPath, "compute_store")
	oldExecutionStorePath := config.Path
	if oldExecutionStorePath == "" {
		oldExecutionStorePath = filepath.Join(oldComputeDir, "executions.db")
	}
	newExecutionStorePath := filepath.Join(oldComputeDir, "state_boltdb.db")
	if err := os.Rename(oldExecutionStorePath, newExecutionStorePath); err != nil {
		return err
	}

	newComputeDir := filepath.Join(repoPath, cfgtypes.ComputeDirName)
	if err := os.Rename(oldComputeDir, newComputeDir); err != nil {
		return err
	}

	oldExecutionDir := filepath.Join(repoPath, "executor_storages")
	newExecutionDir := filepath.Join(newComputeDir, "executions")
	if err := os.Rename(oldExecutionDir, newExecutionDir); err != nil {
		return err
	}
	return nil
}

func migrateOrchestratorStore(repoPath string, config legacy_types.JobStoreConfig) error {
	oldOrchestratorDir := filepath.Join(repoPath, "orchestrator_store")
	oldJobStorePath := config.Path
	if oldJobStorePath == "" {
		oldJobStorePath = filepath.Join(oldOrchestratorDir, "jobs.db")
	}
	newJobStorePath := filepath.Join(oldOrchestratorDir, "state_boltdb.db")
	if err := os.Rename(oldJobStorePath, newJobStorePath); err != nil {
		return err
	}

	newOrchestratorDir := filepath.Join(repoPath, cfgtypes.OrchestratorDirName)
	if err := os.Rename(oldOrchestratorDir, newOrchestratorDir); err != nil {
		return err
	}
	return nil
}
