package repo

const (
	// Version1 is the repo versioning for v1-v1.1.4
	Version1 = iota + 1
	// Version2 is the repo versioning up to v1.2.1
	Version2
	// Version3 is the repo version to (including) v1.4.0
	Version3
	// Version4 is the current repo version
	Version4
	// Add new versions here
	// RepoVersion5
	// RepoVersion6
	// ...
)

// VersionFile is the name of the repo file containing the repo version.
const VersionFile = "repo.version"

// IsValidVersion returns true if the version is valid.
func IsValidVersion(version int) bool {
	return version >= Version1 && version <= Version4
}

type Version struct {
	Version int
}

func (fsr *FsRepo) readVersion() (int, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return -1, err
	}
	return sysmeta.RepoVersion, nil
}
