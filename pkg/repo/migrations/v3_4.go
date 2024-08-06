package migrations

import (
	"os"
	"path/filepath"
	"time"

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
		// migrate values from config to repo.SystemMetadataFile
		{
			_, fileCfg, err := readConfig(r)
			if err != nil {
				return err
			}
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
			repoPath, err := r.Path()
			if err != nil {
				return err
			}

			// delete the update.json file, this has been replaced by repo.SystemMetadataFile
			// ignore errors regarding failure to remove since there isn't a guarantee the file exists
			_ = os.Remove(filepath.Join(repoPath, "update.json"))

			// delete the repo.version file, this has been replaced by repo.SystemMetadataFile.
			return os.Remove(filepath.Join(repoPath, repo.LegacyVersionFile))
		}
	},
)
