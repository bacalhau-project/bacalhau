package ipfs

import (
	"context"
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
	localDir   string
	ipfsClient ipfs.Client
}

func NewStorage(cm *system.CleanupManager, cl ipfs.Client) (*StorageProvider, error) {
	// TODO: consolidate the various config inputs into one package otherwise they are scattered across the codebase
	dir, err := os.MkdirTemp(config.GetStoragePath(), "bacalhau-ipfs")
	if err != nil {
		return nil, err
	}

	cm.RegisterCallback(func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to clean up IPFS storage directory: %w", err)
		}
		return nil
	})

	storageHandler := &StorageProvider{
		ipfsClient: cl,
		localDir:   dir,
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
	ctx, cancel := context.WithTimeout(ctx, config.GetVolumeSizeRequestTimeout())
	defer cancel()

	source, err := DecodeSpec(volume.Source)
	if err != nil {
		return 0, err
	}

	return s.ipfsClient.GetCidSize(ctx, source.CID)
}

func (s *StorageProvider) PrepareStorage(ctx context.Context, storageSpec models.InputSource) (storage.StorageVolume, error) {
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
	volume, err = s.getFileFromIPFS(ctx, source.CID, storageSpec.Target)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to copy %s to volume: %w", storageSpec.Target, err)
	}

	return volume, nil
}

func (s *StorageProvider) CleanupStorage(_ context.Context, storageSpec models.InputSource, _ storage.StorageVolume) error {
	source, err := DecodeSpec(storageSpec.Source)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.localDir, source.CID))
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

func (s *StorageProvider) getFileFromIPFS(ctx context.Context, cid, path string) (storage.StorageVolume, error) {
	outputPath := filepath.Join(s.localDir, cid)

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
		Target: path,
	}

	return volume, nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
