package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const UnknownVersion = -1

const (
	// Version1 is the repo versioning for v1-v1.1.4
	Version1 = iota + 1
	// Version2 is the repo versioning up to v1.2.1
	Version2
	// Version3 is the repo version to (including) v1.4.0
	Version3
	// Version4 is the latest version
	Version4
)

// IsValidVersion returns true if the version is valid.
func IsValidVersion(version int) bool {
	return version >= Version1 && version <= Version4
}

// SystemMetadataFile is the name of the file containing the SystemMetadata.
const SystemMetadataFile = "system_metadata.yaml"

type MetadataStore struct {
	path string
}

func NewMetadataStore(path string) *MetadataStore {
	return &MetadataStore{path: path}
}

func (m *MetadataStore) join(paths ...string) string {
	return filepath.Join(append([]string{m.path}, paths...)...)
}

type SystemMetadata struct {
	RepoVersion     int       `yaml:"RepoVersion"`
	InstallationID  string    `yaml:"InstallationID"`
	InstanceID      string    `yaml:"InstanceID"`
	LastUpdateCheck time.Time `yaml:"LastUpdateCheck"`
	NodeName        string    `yaml:"NodeName"`
}

// WriteVersion updates the RepoVersion in the SystemMetadataFile.
// If the metadata file doesn't exist, it creates a new one.
func (fsr *MetadataStore) WriteVersion(version int) error {
	if version < Version4 {
		return fsr.writeLegacyVersion(version)
	}
	return fsr.updateOrCreateMetadata(func(m *SystemMetadata) {
		m.RepoVersion = version
	})
}

// readVersion reads the RepoVersion in the SystemMetadataFile.
// For repos w/ version <= Version3, the version in repo.version is read.
// If SystemMetadataFile or repo.version doesn't exist, or if their content is invalid, an error is returned.
func (fsr *MetadataStore) readVersion() (int, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		// if the system metadata file does not exist attempt to read the legacy version
		if os.IsNotExist(err) {
			return fsr.readLegacyVersion()
		}
		return UnknownVersion, err
	}
	return sysmeta.RepoVersion, nil
}

// ReadLastUpdateCheck returns the last update check value from system_metadata.yaml
// It fails if the metadata file doesn't exist.
func (fsr *MetadataStore) ReadLastUpdateCheck() (time.Time, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return time.Time{}, err
	}
	return sysmeta.LastUpdateCheck, nil
}

// WriteLastUpdateCheck updates the LastUpdateCheck in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *MetadataStore) WriteLastUpdateCheck(lastUpdateCheck time.Time) error {
	return fsr.updateExistingMetadata(func(m *SystemMetadata) {
		m.LastUpdateCheck = lastUpdateCheck
	})
}

func (fsr *MetadataStore) ReadInstallationID() (string, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return "", err
	}
	return sysmeta.InstallationID, nil
}

// WriteInstallationID updates the InstallationID in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *MetadataStore) WriteInstallationID(id string) error {
	return fsr.updateExistingMetadata(func(sysmeta *SystemMetadata) {
		sysmeta.InstallationID = id
	})
}

// ReadInstanceID reads the InstanceID in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *MetadataStore) ReadInstanceID() (string, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return "", err
	}
	return sysmeta.InstanceID, nil
}

// WriteInstanceID updates the InstanceID in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *MetadataStore) WriteInstanceID(id string) error {
	return fsr.updateExistingMetadata(func(sysmeta *SystemMetadata) {
		sysmeta.InstanceID = id
	})
}

func (fsr *MetadataStore) ReadNodeName() (string, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return "", err
	}
	return sysmeta.NodeName, nil
}

func (fsr *MetadataStore) WriteNodeName(name string) error {
	return fsr.updateExistingMetadata(func(sysmeta *SystemMetadata) {
		sysmeta.NodeName = name
	})
}

// readMetadata unmarshals the content of SystemMedataFile into SystemMetadata and returns it.
// If fails if the metadata file doesn't exist.
func (fsr *MetadataStore) readMetadata() (*SystemMetadata, error) {
	metaBytes, err := os.ReadFile(fsr.join(SystemMetadataFile))
	if err != nil {
		return nil, err
	}
	sysmeta := new(SystemMetadata)
	if err := yaml.Unmarshal(metaBytes, sysmeta); err != nil {
		return nil, fmt.Errorf("unmarshalling repo system metadata: %w", err)
	}
	return sysmeta, nil
}

// writeMetadata marshals the provided SystemMetadata to YAML
// and writes it to the system metadata file, creating the file if it doesn't exist.
func (fsr *MetadataStore) writeMetadata(m *SystemMetadata) error {
	metaBytes, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("`marshaling` repo system metadata: %w", err)
	}

	if err := os.WriteFile(fsr.join(SystemMetadataFile), metaBytes, util.OS_USER_RW); err != nil {
		return fmt.Errorf("writing repo system metadata file: %w", err)
	}

	return nil
}

// updateOrCreateMetadata updates an existing metadata file or creates a new one if it doesn't exist.
// The update is applied using the provided updateFunc.
func (fsr *MetadataStore) updateOrCreateMetadata(updateFunc func(*SystemMetadata)) error {
	filePath := fsr.join(SystemMetadataFile)

	var sysmeta *SystemMetadata

	// Check if the file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// File doesn't exist, create new metadata
		sysmeta = &SystemMetadata{}
	} else if err != nil {
		return fmt.Errorf("checking system metadata file: %w", err)
	} else {
		// File exists, read the current metadata
		sysmeta, err = fsr.readMetadata()
		if err != nil {
			return fmt.Errorf("reading existing metadata: %w", err)
		}
	}

	// Apply the update
	updateFunc(sysmeta)

	// Write the updated metadata back to the file
	return fsr.writeMetadata(sysmeta)
}

// updateExistingMetadata updates an existing metadata file.
// It fails if the file doesn't exist.
// The update is applied using the provided updateFunc.
func (fsr *MetadataStore) updateExistingMetadata(updateFunc func(*SystemMetadata)) error {
	filePath := fsr.join(SystemMetadataFile)

	// Check if the file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("system metadata file does not exist: %w", err)
	} else if err != nil {
		return fmt.Errorf("checking system metadata file: %w", err)
	}

	// File exists, read the current metadata
	currentMetadata, err := fsr.readMetadata()
	if err != nil {
		return fmt.Errorf("reading existing metadata: %w", err)
	}

	// Apply the update
	updateFunc(currentMetadata)

	// Write the updated metadata back to the file
	return fsr.writeMetadata(currentMetadata)
}
