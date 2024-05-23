package repo

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const (
	// RepoVersionLatest is the current repo versioning.
	RepoVersion4 = 4
	// RepoVersion3 is the repo versioning up to v1.3.0
	RepoVersion3 = 3
	// RepoVersion2 is the repo versioning up to v1.2.1
	RepoVersion2 = 2
	// RepoVersion1 is the repo versioning for v1-v1.1.4
	RepoVersion1 = 1
	// RepoVersionFile is the name of the repo file containing the repo version.
	RepoVersionFile = "repo.version"
)

// IsValidVersion returns true if the version is valid.
func IsValidVersion(version int) bool {
	return version >= RepoVersion1 && version <= RepoVersion4
}

type RepoVersion struct {
	Version int
}

func (fsr *FsRepo) writeVersion(version int) error {
	repoVersion := RepoVersion{Version: version}
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
