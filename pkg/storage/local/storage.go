package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

type StorageProviderParams struct {
	AllowedPaths []AllowedPath
}
type StorageProvider struct {
	allowedPaths []AllowedPath
}

func NewStorageProvider(params StorageProviderParams) (*StorageProvider, error) {
	storageHandler := &StorageProvider{
		allowedPaths: params.AllowedPaths,
	}
	log.Debug().Msgf("Local driver created with allowedPaths: %s", storageHandler.allowedPaths)

	return storageHandler, nil
}

func (driver *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return len(driver.allowedPaths) > 0, nil
}

func (driver *StorageProvider) HasStorageLocally(_ context.Context, volume models.InputSource) (bool, error) {
	source, err := DecodeSpec(volume.Source)
	if err != nil {
		return false, err
	}

	if _, err = driver.matchAllowedPath(source); err != nil {
		return false, nil
	}

	if _, err := os.Stat(source.SourcePath); errors.Is(err, os.ErrNotExist) {
		// If the volume does not exist, we should look into whether we can create it.
		// In order to be able to create it, ReadWrite must be true and CreateAs should be permissive.
		return source.ReadWrite && source.CreateAs != NoCreate, nil
	}
	return true, nil
}

func (driver *StorageProvider) GetVolumeSize(_ context.Context, _ *models.Execution, volume models.InputSource) (uint64, error) {
	source, err := DecodeSpec(volume.Source)
	if err != nil {
		return 0, err
	}

	_, err = driver.matchAllowedPath(source)
	if err != nil {
		return 0, err
	}

	// check if the volume exists
	if _, err := os.Stat(source.SourcePath); errors.Is(err, os.ErrNotExist) {
		// if the volume does not exist but we can create it, we should return 0.
		// we only know that it's going to be an empty file/directory at the start of the job
		if source.ReadWrite && source.CreateAs != NoCreate {
			return 0, nil
		}

		// if the volume does not exist and we can't create it, return an error
		return 0, bacerrors.Newf("volume %s does not exist and creation is not allowed", source.SourcePath).
			WithHint("If you want the job to create the volume, set ReadWrite to true and CreateAs to one of [%s]",
				strings.Join(PermissiveCreateStrategies(), ", "))
	}
	// We only query the volume size to make sure we have enough disk space to pull mount the volume locally from a remote location.
	// In this case the data is already local and attempting to query the size would be a waste of time.
	return 0, nil
}

func (driver *StorageProvider) PrepareStorage(
	_ context.Context,
	_ string,
	_ *models.Execution,
	input models.InputSource,
) (storage.StorageVolume, error) {
	source, err := DecodeSpec(input.Source)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	_, err = driver.matchAllowedPath(source)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	err = driver.createVolumeIfNotExists(source)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	return storage.StorageVolume{
		Type:     storage.StorageVolumeConnectorBind,
		ReadOnly: !source.ReadWrite,
		Source:   source.SourcePath,
		Target:   input.Target,
	}, nil
}

func (driver *StorageProvider) CleanupStorage(_ context.Context, _ models.InputSource, _ storage.StorageVolume) error {
	// We should NOT clean up the storage as it is a locally mounted volume.
	// We are mounting the source directory directly to the target directory and not copying the data.
	// Even if we created the directory/file, we keep it
	return nil
}

func (driver *StorageProvider) Upload(context.Context, string) (models.SpecConfig, error) {
	return models.SpecConfig{}, fmt.Errorf("not implemented")
}

func (driver *StorageProvider) matchAllowedPath(storageSpec Source) (*AllowedPath, error) {
	var insufficientPermissions bool

	for _, driverAllowedPath := range driver.allowedPaths {
		match, err := doublestar.PathMatch(driverAllowedPath.Path, storageSpec.SourcePath)
		if match && err == nil {
			// If storage wants read-write access but driver only allows read-only,
			// set a flag indicating there was a match but permissions are wrong.
			if storageSpec.ReadWrite && !driverAllowedPath.ReadWrite {
				insufficientPermissions = true
				continue
			}

			// Found a match and the permissions are correct
			return &driverAllowedPath, nil
		}
	}

	var err bacerrors.Error
	if insufficientPermissions {
		err = bacerrors.Newf("volume %s is not granted write access", storageSpec.SourcePath)
	} else {
		err = bacerrors.Newf("volume %s is not allowlisted", storageSpec.SourcePath)
	}
	err = err.WithCode(bacerrors.ConfigurationError).
		WithHint("Verify Compute.AllowListedLocalPaths configuration property")
	return nil, err
}

func (driver *StorageProvider) createVolumeIfNotExists(source Source) error {
	_, err := os.Stat(source.SourcePath)
	if err == nil {
		// The volume exists, do nothing
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		// Something else went wrong when accessing the volume (for example permission denied) return an error
		return bacerrors.Wrapf(err, "could not access volume at %s", source.SourcePath)
	}

	// Can only create if the job spec allows read-write access
	if !source.ReadWrite {
		return bacerrors.Newf("volume %s does not exist and read-write access is not allowed", source.SourcePath).
			WithHint("If you want the job to create the volume, set the input Source ReadWrite property to true")
	}

	switch source.CreateAs {
	case NoCreate:
		// the volume does not exist and we can't create it, return error
		return bacerrors.Newf("volume does not exist at %s and creation is not allowed", source.SourcePath).
			WithHint(fmt.Sprintf("If you want the job to create the volume, set the CreateAs property to either '%s' or '%s'", Dir, File))
	case Dir:
		err := os.MkdirAll(source.SourcePath, util.OS_USER_RWX)
		if err != nil {
			return bacerrors.Wrapf(err, "could not create source directory at %s", source.SourcePath)
		}
	case File:
		return createEmptyFile(source.SourcePath)
	default:
		// this should never happen, but catch an error in case a new CreateStrategy is added but not handled here
		return bacerrors.Newf("unknown CreateAs value %s", source.CreateAs)
	}
	return nil
}

func createEmptyFile(filePath string) error {
	dir, file := filepath.Split(filePath)

	// Edge case: SourcePath points to a directory but CreateAs is set to File.
	// Likely an error in the job spec, return an error.
	if file == "" {
		return bacerrors.Newf("SourcePath %s is a directory but CreateAs is set to %s", filePath, File.String()).
			WithHint("Change the SourcePath to point to a file or set CreateAs to %s", Dir.String())
	}

	//create parent directory
	var err error
	if _, err = os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(dir, util.OS_USER_RWX)
	}
	if err != nil {
		return bacerrors.Wrapf(err, "could not create source directory at %s", dir)
	}

	// create the file
	//nolint:gosec // G304: filePath from storage source, validated
	fileHandle, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_RDONLY, util.OS_USER_RWX)
	if err != nil {
		return bacerrors.Wrapf(err, "could not create source file at %s", filePath)
	}
	_ = fileHandle.Close()
	return nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
