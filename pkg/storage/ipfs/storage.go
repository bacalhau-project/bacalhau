package ipfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// a storage driver runs the downloads content
// from a remote ipfs server and copies it to
// to a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	ipfsClient ipfs.Client
}

func NewStorage(cl ipfs.Client) (*StorageProvider, error) {
	storageHandler := &StorageProvider{
		ipfsClient: cl,
	}

	log.Trace().Msgf("IPFS API Copy driver created with address: %s", cl.APIAddress())
	return storageHandler, nil
}

func (s *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	_, err := s.ipfsClient.ID(ctx)
	return err == nil, err
}

func (s *StorageProvider) HasStorageLocally(ctx context.Context, volume models.InputSource) (bool, error) {
	source, err := DecodeSpec(volume.Source)
	if err != nil {
		return false, err
	}
	return s.ipfsClient.HasCID(ctx, source.CID)
}

func (s *StorageProvider) GetVolumeSize(ctx context.Context, volume models.InputSource) (uint64, error) {
	// we wrap this in a timeout because if the CID is not present on the network this seems to hang

	// TODO(forrest) [correctness] this timeout should be passed in as a param or set on the context by the method caller.
	// for further context on why this is the way it is see: https://github.com/bacalhau-project/bacalhau/pull/1432
	timeoutDuration := config.GetVolumeSizeRequestTimeout()
	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	source, err := DecodeSpec(volume.Source)
	if err != nil {
		return 0, err
	}

	size, err := s.ipfsClient.GetCidSize(ctx, source.CID)
	if err != nil {
		// we failed to find the content before the context timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("IPFS storage provider was unable to retrieve content %q before timeout %s: %w",
				source.CID,
				timeoutDuration, err)
		}
		return 0, fmt.Errorf("IPFS storage provider was unable to retrieve content %q: %w", source.CID, err)
	}
	return size, nil
}

func (s *StorageProvider) PrepareStorage(
	ctx context.Context,
	storageDirectory string,
	storageSpec models.InputSource) (storage.StorageVolume, error) {
	source, err := DecodeSpec(storageSpec.Source)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	stat, err := s.ipfsClient.Stat(ctx, source.CID)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to stat %s: %w", source.CID, err)
	}

	if stat.Type != ipfs.IPLDFile && stat.Type != ipfs.IPLDDirectory {
		return storage.StorageVolume{}, fmt.Errorf("unknown ipld file type for %s: %v", source.CID, stat.Type)
	}

	var volume storage.StorageVolume
	volume, err = s.getFileFromIPFS(ctx, source.CID, storageDirectory, storageSpec.Target)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to copy %s to volume: %w", storageSpec.Target, err)
	}

	return volume, nil
}

func (s *StorageProvider) CleanupStorage(_ context.Context, storageSpec models.InputSource, vol storage.StorageVolume) error {
	fileInfo, err := os.Stat(vol.Source)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return os.RemoveAll(vol.Source)
	}

	return os.Remove(vol.Source)
}

func (s *StorageProvider) Upload(ctx context.Context, localPath string) (models.SpecConfig, error) {
	cid, err := s.ipfsClient.Put(ctx, localPath)
	if err != nil {
		return models.SpecConfig{}, err
	}

	return models.SpecConfig{
		Type: models.StorageSourceIPFS,
		Params: Source{
			CID: cid,
		}.ToMap(),
	}, nil
}

func (s *StorageProvider) getFileFromIPFS(ctx context.Context, cid, storageDirectory, mountPath string) (storage.StorageVolume, error) {
	outputPath := filepath.Join(storageDirectory, cid)

	// If the output path already exists, we already have the data, as
	// ipfsClient.Get(...) renames the result path atomically after it has
	// finished downloading the CID.
	ok, err := system.PathExists(outputPath)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	if !ok {
		err = s.ipfsClient.Get(ctx, cid, outputPath)
		if err != nil {
			return storage.StorageVolume{}, err
		}
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputPath,
		Target: mountPath,
	}

	return volume, nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
