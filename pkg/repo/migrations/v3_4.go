package migrations

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

var V3Migration = repo.NewMigration(
	repo.Version3,
	repo.Version4,
	func(r repo.FsRepo) error {
		repoPath, err := r.Path()
		if err != nil {
			return err
		}
		// read the old version file
		versionPath := filepath.Join(repoPath, repo.LegacyVersionFile)
		versionBytes, err := os.ReadFile(versionPath)
		if err != nil {
			return err
		}
		var version repo.Version
		if err := json.Unmarshal(versionBytes, &version); err != nil {
			return err
		}

		// write version 4 to the system_metadata.yaml file
		if err := r.WriteVersion(repo.Version4); err != nil {
			return err
		}

		_, fileCfg, err := readConfig(r)
		if err != nil {
			return err
		}
		if fileCfg.User.InstallationID != "" {
			if err := r.WriteInstallationID(fileCfg.User.InstallationID); err != nil {
				return err
			}
		}

		// delete the update.json file, this has been replaced by repo.SystemMetadataFile
		// ignore errors regarding failure to remove since there isn't a guarantee the file exists
		_ = os.Remove(filepath.Join(repoPath, "update.json"))

		// delete the old version file
		return os.Remove(versionPath)
	},
)
