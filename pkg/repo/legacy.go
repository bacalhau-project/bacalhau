package repo

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

// LegacyVersionFile is the name of the repo file containing the repo version.
// This file is only used in versions 1.4.0 and below. Subsequent bacalhau
// versions persist the repo version to system_metadata.yaml
const LegacyVersionFile = "repo.version"

type Version struct {
	Version int
}

// readLegacyVersion reads the repo version from the LegacyVersionFile.
func (fsr *MetadataStore) readLegacyVersion() (int, error) {
	versionPath := fsr.join(LegacyVersionFile)
	versionBytes, err := os.ReadFile(versionPath)
	if err != nil {
		return UnknownVersion, err
	}
	var version Version
	if err := json.Unmarshal(versionBytes, &version); err != nil {
		return -1, err
	}
	if !IsValidVersion(version.Version) {
		return -1, NewUnknownRepoVersionError(version.Version)
	}
	return version.Version, nil
}

// writeLegacyVersion writes the repo version to LegacyVersionFile.
func (fsr *MetadataStore) writeLegacyVersion(version int) error {
	repoVersion := Version{Version: version}
	versionJSON, err := json.Marshal(repoVersion)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(fsr.path, LegacyVersionFile), versionJSON, util.OS_USER_RW)
}
