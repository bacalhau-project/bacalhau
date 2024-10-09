package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
	legacy_types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
var V3Migration = V3MigrationWithConfig(system.DefaultGlobalConfig)

func V3MigrationWithConfig(globalCfg system.GlobalConfig) repo.Migration {
	return repo.NewMigration(
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
				// update the legacy version file so older versions fail gracefully.
				if err := r.WriteLegacyVersion(repo.Version4); err != nil {
					return fmt.Errorf("updating repo.version: %w", err)
				}
				if err := r.WriteLastUpdateCheck(time.UnixMilli(0)); err != nil {
					return err
				}

				// ignore this error as the file may not exist
				_ = os.Remove(filepath.Join(repoPath, "update.json"))
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
					// migrate installationID if none is present in the system wide config
					if fileCfg.User.InstallationID != "" && globalCfg.InstallationID() == "" {
						if err := writeInstallationID(globalCfg, fileCfg.User.InstallationID); err != nil {
							return err
						}
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

					// Write the updated config back to the same file
					newConfigBytes, err := yaml.Marshal(&newConfigType)
					if err != nil {
						return fmt.Errorf("marshaling new config: %w", err)
					}
					if err := os.WriteFile(oldConfigFilePath, newConfigBytes, util.OS_USER_RWX); err != nil {
						return fmt.Errorf("writing updated config file: %w", err)
					}
				} else if !os.IsNotExist(err) {
					// if there was an error other than the file not existing, abort.
					return fmt.Errorf("failed to read config file %s while migrating: %w", oldConfigFilePath, err)
				}

			}
			return nil
		},
	)
}

func migrateComputeStore(repoPath string, config legacy_types.JobStoreConfig) error {
	oldComputeDir := filepath.Join(repoPath, "compute_store")
	oldExecutionStorePath := config.Path
	if oldExecutionStorePath == "" {
		oldExecutionStorePath = filepath.Join(oldComputeDir, "executions.db")
	}
	// if a store is present migrate it.
	if _, err := os.Stat(oldExecutionStorePath); err == nil {
		newExecutionStorePath := filepath.Join(oldComputeDir, "state_boltdb.db")
		if err := os.Rename(oldExecutionStorePath, newExecutionStorePath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	newComputeDir := filepath.Join(repoPath, types.ComputeDirName)
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
	// if a store is present migrate it.
	if _, err := os.Stat(oldJobStorePath); err == nil {
		newJobStorePath := filepath.Join(oldOrchestratorDir, "state_boltdb.db")
		if err := os.Rename(oldJobStorePath, newJobStorePath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	newOrchestratorDir := filepath.Join(repoPath, types.OrchestratorDirName)
	if err := os.Rename(oldOrchestratorDir, newOrchestratorDir); err != nil {
		return err
	}
	return nil
}
