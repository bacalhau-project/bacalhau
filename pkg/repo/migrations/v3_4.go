package migrations

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// V3Migration updates the repo replacing repo.version and update.json with system_metadata.yaml.
// It does the following:
// - creates system_metadata.yaml with repo version 4.
// - sets the last update check time in system_metadata.yaml to unix time zero.
// - if an installationID is present in the config its value is persisted to system_metadata.yaml.
// - removes update.json if the file is present.
// - removes repo.version.
var V3Migration = repo.NewMigration(
	repo.Version3,
	repo.Version4,
	func(r repo.FsRepo) error {
		repoPath, err := r.Path()
		if err != nil {
			return fmt.Errorf("getting repo path: %w", err)
		}

		_, fileCfg, err := readConfig(r)
		if err != nil {
			return fmt.Errorf("reading config from repo: %w", err)
		}

		// Create a temporary staging directory, in the event a migration step fails we don't fubar the repo.
		stagingPath := filepath.Join(os.TempDir(), "bacalhau-migration-staging")
		if err := os.Mkdir(stagingPath, os.ModePerm); err != nil {
			return fmt.Errorf("creating staging directory: %w", err)
		}
		defer func() {
			if err := os.RemoveAll(stagingPath); err != nil {
				log.Warn().Err(err).Str("path", stagingPath).Msg("failed to clean up staging path for migration.")
			}
		}() // Clean up staging directory regardless of failure

		// Stage all changes in the temporary directory
		if err := stageMigration(r, fileCfg, stagingPath); err != nil {
			return fmt.Errorf("staging migration: %w", err)
		}

		// Commit the changes by moving the staged directory to the actual location
		if err := commitMigration(stagingPath, repoPath); err != nil {
			return fmt.Errorf("committing migration: %w", err)
		}

		return nil
	},
)

func stageMigration(r repo.FsRepo, fileCfg types.BacalhauConfig, stagingPath string) error {
	repoPath, err := r.Path()
	if err != nil {
		return err
	}
	// create a staging repo to run the migration in
	stagingRepo, err := repo.NewFS(repo.FsRepoParams{
		Path:       stagingPath,
		Migrations: nil,
	})
	if err != nil {
		return err
	}
	// copy the current repo into the staging repo
	if err := copyFS(stagingPath, os.DirFS(repoPath)); err != nil {
		return err
	}
	// from this point, all operations are done on a staging bacalhau repo
	// Initialize the SystemMetadataFile in the staging directory
	if err := stagingRepo.WriteVersion(repo.Version4); err != nil {
		return err
	}
	if err := stagingRepo.WriteLastUpdateCheck(time.UnixMilli(0)); err != nil {
		return err
	}
	if fileCfg.User.InstallationID != "" {
		if err := stagingRepo.WriteInstallationID(fileCfg.User.InstallationID); err != nil {
			return err
		}
	}

	// Copy or move files as needed, but in the staging directory
	if err := migrateLegacyFiles(stagingPath); err != nil {
		return err
	}
	if err := migrateRepoPaths(fileCfg, stagingPath, repoPath); err != nil {
		return err
	}

	return nil
}

func migrateLegacyFiles(path string) error {
	// Delete or move legacy files, but within the staging directory
	_ = os.Remove(filepath.Join(path, "update.json"))
	if err := os.Remove(filepath.Join(path, repo.LegacyVersionFile)); err != nil {
		return fmt.Errorf("removing legacy repo version file: %w", err)
	}
	return nil
}

func migrateRepoPaths(fileCfg types.BacalhauConfig, stagingPath, repoPath string) error {
	// Stage migration of paths
	if fileCfg.User.KeyPath != "" && fileCfg.User.KeyPath != filepath.Join(repoPath, repo.UserKeyFile) {
		if err := stageFileCopy(fileCfg.User.KeyPath, filepath.Join(stagingPath, repo.UserKeyFile)); err != nil {
			return fmt.Errorf("copying user key file: %w", err)
		}
	}

	if fileCfg.Node.Compute.ExecutionStore.Path != "" &&
		fileCfg.Node.Compute.ExecutionStore.Path != filepath.Join(repoPath, repo.ComputeDirKey, "executions.db") {
		if err := stageFileCopy(
			fileCfg.Node.Compute.ExecutionStore.Path,
			filepath.Join(stagingPath, repo.ComputeDirKey, "executions.db"),
		); err != nil {
			return fmt.Errorf("copying execution database: %w", err)
		}
	}

	if fileCfg.Node.Requester.JobStore.Path != "" &&
		fileCfg.Node.Requester.JobStore.Path != filepath.Join(repoPath, repo.OrchestratorDirKey, "jobs.db") {
		if err := stageFileCopy(
			fileCfg.Node.Requester.JobStore.Path,
			filepath.Join(stagingPath, repo.OrchestratorDirKey, "jobs.db"),
		); err != nil {
			return fmt.Errorf("copying job database: %w", err)
		}
	}

	from := fileCfg.Node.ComputeStoragePath
	if from == "" {
		from = filepath.Join(repoPath, "executor_storages")
	}
	if err := copyFS(filepath.Join(stagingPath, repo.ExecutionDirKey), os.DirFS(from)); err != nil {
		return fmt.Errorf("copying executor storages: %w", err)
	}

	return nil
}

func stageFileCopy(from, to string) error {
	log.Info().Msgf("staging file copy from %q to %q", from, to)
	return copyFile(from, to)
}

func commitMigration(stagingPath, repoPath string) error {
	// Replace the original paths with the staged changes
	log.Info().Msgf("committing staged changes from %q to %q", stagingPath, repoPath)
	// create a backup of the repo pre-migration
	backupPath := filepath.Join(filepath.Dir(repoPath), fmt.Sprintf(".bacalhau_backup_%d", time.Now().Unix()))
	// move the current bacalhau repo into the backup
	if err := os.Rename(repoPath, backupPath); err != nil {
		return err
	}
	// rename move the migrated repo to the specified repo path
	// this is the actual migration
	if err := os.Rename(stagingPath, repoPath); err != nil {
		return err
	}
	// rename the backup if the above step was successful
	return os.RemoveAll(backupPath)
}

// copyFS copies the file system fsys into the directory dir,
// creating dir if necessary.
//
// Newly created directories and files have their default modes
// according to the current umask, except that the execute bits
// are copied from the file in fsys when creating a local file.
//
// If a file name in fsys does not satisfy filepath.IsLocal,
// an error is returned for that file.
//
// Copying stops at and returns the first error encountered.
// Credit: https://github.com/golang/go/issues/62484
func copyFS(dir string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Handle the error before accessing d
			return fmt.Errorf("error accessing %s: %v", path, err)
		}

		targ := filepath.Join(dir, filepath.FromSlash(path))

		if d.IsDir() {
			// Create the directory
			dinfor, err := d.Info()
			if err != nil {
				return fmt.Errorf("stating directory: %w", err)
			}
			if err := os.MkdirAll(targ, dinfor.Mode()); err != nil {
				return fmt.Errorf("creating directory %s: %v", targ, err)
			}
			return nil
		}

		// Open the file in the fs.FS
		r, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %v", path, err)
		}
		defer r.Close()

		// Get file info to copy the mode
		info, err := r.Stat()
		if err != nil {
			return fmt.Errorf("getting file info for %s: %v", path, err)
		}

		// Create the destination file with the same permissions
		w, err := os.OpenFile(targ, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return fmt.Errorf("creating file %s: %v", targ, err)
		}

		// Copy the file contents
		if _, err := io.Copy(w, r); err != nil {
			w.Close() // ensure the file is closed even if there's an error
			return fmt.Errorf("copying %s: %v", path, err)
		}

		// Close the destination file
		if err := w.Close(); err != nil {
			return fmt.Errorf("closing file %s: %v", targ, err)
		}

		return nil
	})
}

func copyFile(srcPath, dstPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get the file info of the source file to retrieve its permissions
	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Create the destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Set the destination file's permissions to match the source file's permissions
	err = os.Chmod(dstPath, srcFileInfo.Mode())
	if err != nil {
		return err
	}

	// Flush the contents to disk
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
