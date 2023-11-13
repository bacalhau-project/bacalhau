package repo

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const (
	// RepoVersion1 is the current repo versioning.
	RepoVersion1 = 1
	// RepoVersionFile is the name of the repo file containing the repo version.
	RepoVersionFile = "repo.version"
)

type RepoVersion struct {
	Version int
}

func (fsr *FsRepo) writeVersion() error {
	repoVersion := RepoVersion{Version: RepoVersion1}
	versionJSON, err := json.Marshal(repoVersion)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(fsr.path, RepoVersionFile), versionJSON, util.OS_USER_RW)
}

func (fsr *FsRepo) readVersion() (int, error) {
	versionBytes, err := os.ReadFile(filepath.Join(fsr.path, RepoVersionFile))
	if err != nil {
		return -1, err
	}
	var version RepoVersion
	if err := json.Unmarshal(versionBytes, &version); err != nil {
		return -1, err
	}
	return version.Version, nil
}
