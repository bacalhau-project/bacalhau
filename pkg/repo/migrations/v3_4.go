package migrations

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
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

		// migrate values from config to repo.SystemMetadataFile
		{
			// create the SystemMetadataFile
			{
				// initialize the repo.SystemMetadataFile by writing the version to it.
				if err := r.WriteVersion(repo.Version4); err != nil {
					return err
				}
				// reset the last update check to zero.
				if err := r.WriteLastUpdateCheck(time.UnixMilli(0)); err != nil {
					return err
				}
				// if an installationID is present migrate it.
				if fileCfg.User.InstallationID != "" {
					if err := r.WriteInstallationID(fileCfg.User.InstallationID); err != nil {
						return err
					}
				}
			}

			// remove legacy repo files.
			{
				// delete the update.json file, this has been replaced by repo.SystemMetadataFile
				// ignore errors regarding failure to remove since there isn't a guarantee the file exists
				_ = os.Remove(filepath.Join(repoPath, "update.json"))

				// delete the repo.version file, this has been replaced by repo.SystemMetadataFile.
				if err := os.Remove(filepath.Join(repoPath, repo.LegacyVersionFile)); err != nil {
					return fmt.Errorf("removing legacy repo version file: %w", err)
				}
			}
		}

		// migrate new repo paths
		// TODO migrate the job and execution databases
		{
			if fileCfg.User.KeyPath != "" && fileCfg.User.KeyPath != filepath.Join(repoPath, repo.UserKeyFile) {
				// the user has configured a non-standard location for their private key file, copy it to the repo.
				from := fileCfg.User.KeyPath
				to := filepath.Join(repoPath, repo.UserKeyFile)
				log.Info().Msgf("copying user key from %q to %q", from, to)
				if err := copyFile(from, to); err != nil {
					return fmt.Errorf("copying user key file: %w", err)
				}
				log.Info().Msgf("copied user key from %q to %q. You may discard %q", from, to, from)
			}

			// this is actually 'executor_storages', we need to move it under 'compute_store'
			// compute_store doesn't have a configurable path, thankfully
			from := fileCfg.Node.ComputeStoragePath
			if from == "" {
				from = filepath.Join(repoPath, "executor_storages")
			}
			fromFS := os.DirFS(from)
			to := filepath.Join(repoPath, repo.ExecutionDirKey)
			log.Info().Msgf("copying execution dir from %q to %q", from, to)
			// copy the contents from ~/.bacalhau/executor_storages/* to  ~/.bacalhau/compute_store/executions/*
			if err := copyFS(to, fromFS); err != nil {
				return fmt.Errorf("copying executor_storages: %w", err)
			}
			log.Info().Msgf("copied execution dir from %q to %q. You may discard %q", from, to, from)
		}
		return nil
	},
)

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
			if err := os.MkdirAll(targ, util.OS_USER_RW); err != nil {
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
