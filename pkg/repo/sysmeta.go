package repo

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

type SystemMetadata struct {
	RepoVersion     int       `yaml:"Version"`
	InstallationID  string    `yaml:"InstallationID"`
	LastUpdateCheck time.Time `yaml:"LastUpdateCheck"`
}

const SystemMetadataFile = "system_metadata.yaml"

// WriteVersion updates the Version in the metadata.
// If the metadata file doesn't exist, it creates a new one.
func (fsr *FsRepo) WriteVersion(version int) error {
	if version < Version4 {
		return fsr.writeLegacyVersion(version)
	}
	return fsr.updateOrCreateMetadata(func(m *SystemMetadata) {
		m.RepoVersion = version
	})
}

func (fsr *FsRepo) ReadLastUpdateCheck() (time.Time, error) {
	repoMeta, err := fsr.readMetadata()
	if err != nil {
		return time.Time{}, err
	}
	return repoMeta.LastUpdateCheck, nil
}

// WriteLastUpdateCheck updates the LastUpdateCheck in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *FsRepo) WriteLastUpdateCheck(lastUpdateCheck time.Time) error {
	return fsr.updateExistingMetadata(func(m *SystemMetadata) {
		m.LastUpdateCheck = lastUpdateCheck
	})
}

// WriteInstallationID updates the InstallationID in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *FsRepo) WriteInstallationID(id string) error {
	return fsr.updateExistingMetadata(func(metadata *SystemMetadata) {
		metadata.InstallationID = id
	})
}

// readMetadata opens the system_metadata.yaml file in the repo, returning the result.
func (fsr *FsRepo) readMetadata() (*SystemMetadata, error) {
	metaBytes, err := os.ReadFile(fsr.join(SystemMetadataFile))
	if err != nil {
		return nil, fmt.Errorf("reading repo system metadata file: %w", err)
	}
	metadata := new(SystemMetadata)
	if err := yaml.Unmarshal(metaBytes, metadata); err != nil {
		return nil, fmt.Errorf("unmarshalling repo system metadata: %w", err)
	}
	return metadata, nil
}

// updateOrCreateMetadata updates an existing metadata file or creates a new one if it doesn't exist.
// The update is applied using the provided updateFunc.
func (fsr *FsRepo) updateOrCreateMetadata(updateFunc func(*SystemMetadata)) error {
	filePath := fsr.join(SystemMetadataFile)

	var metadata *SystemMetadata

	// Check if the file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// File doesn't exist, create new metadata
		metadata = &SystemMetadata{}
	} else if err != nil {
		return fmt.Errorf("checking system metadata file: %w", err)
	} else {
		// File exists, read the current metadata
		metadata, err = fsr.readMetadata()
		if err != nil {
			return fmt.Errorf("reading existing metadata: %w", err)
		}
	}

	// Apply the update
	updateFunc(metadata)

	// Write the updated metadata back to the file
	return fsr.writeMetadata(metadata)
}

// updateExistingMetadata updates an existing metadata file.
// It fails if the file doesn't exist.
// The update is applied using the provided updateFunc.
func (fsr *FsRepo) updateExistingMetadata(updateFunc func(*SystemMetadata)) error {
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

// writeMetadata marshals the provided SystemMetadata to YAML
// and writes it to the system metadata file.
func (fsr *FsRepo) writeMetadata(m *SystemMetadata) error {
	metaBytes, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("`marshaling` repo system metadata: %w", err)
	}

	if err := os.WriteFile(fsr.join(SystemMetadataFile), metaBytes, util.OS_USER_RW); err != nil {
		return fmt.Errorf("writing repo system metadata file: %w", err)
	}

	return nil
}
