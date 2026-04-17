package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

const UnknownVersion = -1

const (
	// Version4 is the latest version starting from v1.5.0
	Version4 = 4
	// Version5 adds CLI profiles for managing multiple Bacalhau clusters
	Version5 = 5
)

// IsValidVersion returns true if the version is valid.
func IsValidVersion(version int) bool {
	return version >= Version4 && version <= Version5
}

// SystemMetadataFile is the name of the file containing the SystemMetadata.
const SystemMetadataFile = "system_metadata.yaml"

type SystemMetadata struct {
	RepoVersion     int       `yaml:"RepoVersion"`
	InstanceID      string    `yaml:"InstanceID"`
	LastUpdateCheck time.Time `yaml:"LastUpdateCheck"`
	NodeName        string    `yaml:"NodeName"`
}

func LoadSystemMetadata(path string) (*SystemMetadata, error) {
	metaPath := filepath.Join(path, SystemMetadataFile)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("system metadata doesn't exist at path %s: %w", metaPath, err)
	} else if err != nil {
		return nil, fmt.Errorf("failed to read system metadata at path %s: %w", metaPath, err)
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	sysmeta := new(SystemMetadata)
	if err := yaml.Unmarshal(metaBytes, sysmeta); err != nil {
		return nil, fmt.Errorf("unmarshalling repo system metadata: %w", err)
	}
	return sysmeta, nil
}

func (fsr *FsRepo) SystemMetadata() (*SystemMetadata, error) {
	if exists, err := fsr.Exists(); !exists {
		return nil, fmt.Errorf("cannot read system metadata. repo uninitalized")
	} else if err != nil {
		return nil, fmt.Errorf("opening repo system metadata: %w", err)
	}
	return fsr.readMetadata()
}

// WriteVersion updates the RepoVersion in the SystemMetadataFile.
// If the metadata file doesn't exist, it creates a new one.
func (fsr *FsRepo) WriteVersion(version int) error {
	return fsr.updateOrCreateMetadata(func(m *SystemMetadata) {
		m.RepoVersion = version
	})
}

// readVersion reads the RepoVersion in the SystemMetadataFile.
// If SystemMetadataFile doesn't exist, or if their content is invalid, an error is returned.
func (fsr *FsRepo) readVersion() (int, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return UnknownVersion, err
	}
	return sysmeta.RepoVersion, nil
}

// ReadLastUpdateCheck returns the last update check value from system_metadata.yaml
// It fails if the metadata file doesn't exist.
func (fsr *FsRepo) ReadLastUpdateCheck() (time.Time, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return time.Time{}, err
	}
	return sysmeta.LastUpdateCheck, nil
}

// WriteLastUpdateCheck updates the LastUpdateCheck in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *FsRepo) WriteLastUpdateCheck(lastUpdateCheck time.Time) error {
	return fsr.updateExistingMetadata(func(m *SystemMetadata) {
		m.LastUpdateCheck = lastUpdateCheck
	})
}

// InstanceID reads the InstanceID in the metadata.
// It returns empty string if it fails to read the metadata.
func (fsr *FsRepo) InstanceID() string {
	instanceID, err := fsr.MustInstanceID()
	if err != nil {
		log.Debug().Err(err).Msg("failed to read instanceID")
		return ""
	}
	return instanceID
}

// MustInstanceID reads the InstanceID in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *FsRepo) MustInstanceID() (string, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return "", err
	}
	return sysmeta.InstanceID, nil
}

// writeInstanceID updates the InstanceID in the metadata.
// It fails if the metadata file doesn't exist.
func (fsr *FsRepo) writeInstanceID(id string) error {
	return fsr.updateExistingMetadata(func(sysmeta *SystemMetadata) {
		sysmeta.InstanceID = id
	})
}

func (fsr *FsRepo) ReadNodeName() (string, error) {
	sysmeta, err := fsr.readMetadata()
	if err != nil {
		return "", err
	}
	return sysmeta.NodeName, nil
}

func (fsr *FsRepo) WriteNodeName(name string) error {
	return fsr.updateExistingMetadata(func(sysmeta *SystemMetadata) {
		sysmeta.NodeName = name
	})
}

// readMetadata unmarshals the content of SystemMedataFile into SystemMetadata and returns it.
// If fails if the metadata file doesn't exist.
func (fsr *FsRepo) readMetadata() (*SystemMetadata, error) {
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

// updateOrCreateMetadata updates an existing metadata file or creates a new one if it doesn't exist.
// The update is applied using the provided updateFunc.
func (fsr *FsRepo) updateOrCreateMetadata(updateFunc func(*SystemMetadata)) error {
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
