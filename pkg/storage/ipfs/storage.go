package ipfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
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

func (s *StorageProvider) HasStorageLocally(ctx context.Context, volume spec.Storage) (bool, error) {
	ipfsspec, err := spec_ipfs.Decode(volume)
	if err != nil {
		return false, err
	}
	return s.ipfsClient.HasCID(ctx, ipfsspec.CID.String())
}

func (s *StorageProvider) GetVolumeSize(ctx context.Context, volume spec.Storage) (uint64, error) {
	ipfsspec, err := spec_ipfs.Decode(volume)
	if err != nil {
		return 0, err
	}

	// we wrap this in a timeout because if the CID is not present on the network this seems to hang
	ctx, cancel := context.WithTimeout(ctx, config.GetVolumeSizeRequestTimeout(ctx))
	defer cancel()

	return s.ipfsClient.GetCidSize(ctx, ipfsspec.CID.String())
}

func (s *StorageProvider) PrepareStorage(ctx context.Context, storageSpec spec.Storage) (storage.StorageVolume, error) {
	ipfsspec, err := spec_ipfs.Decode(storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	stat, err := s.ipfsClient.Stat(ctx, ipfsspec.CID.String())
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to stat %s: %w", ipfsspec.CID, err)
	}

	if stat.Type != ipfs.IPLDFile && stat.Type != ipfs.IPLDDirectory {
		return storage.StorageVolume{}, fmt.Errorf("unknown ipld file type for %s: %v", ipfsspec.CID, stat.Type)
	}

	var volume storage.StorageVolume
	volume, err = s.getFileFromIPFS(ctx, storageSpec)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to copy %s to volume: %w", storageSpec.Mount, err)
	}

	return volume, nil
}

func (s *StorageProvider) CleanupStorage(_ context.Context, storageSpec spec.Storage, _ storage.StorageVolume) error {
	ipfsspec, err := spec_ipfs.Decode(storageSpec)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.localDir, ipfsspec.CID.String()))
}

func (s *StorageProvider) Upload(ctx context.Context, localPath string) (spec.Storage, error) {
	c, err := s.ipfsClient.Put(ctx, localPath)
	if err != nil {
		return spec.Storage{}, err
	}
	specCid, err := cid.Decode(c)
	if err != nil {
		return spec.Storage{}, err
	}
	// FIXME(frrist): constructing a somewhat invalid spec here, we need a name and a mount path
	return (&spec_ipfs.IPFSStorageSpec{CID: specCid}).AsSpec("TODO", "TODO")
}

// TODO we could pass an IpfsStorageSpec and a mount path here instead to avoid an extra decoding since
// the caller of this method has already decoded the storage spec
func (s *StorageProvider) getFileFromIPFS(ctx context.Context, storageSpec spec.Storage) (storage.StorageVolume, error) {
	ipfsspec, err := spec_ipfs.Decode(storageSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	outputPath := filepath.Join(s.localDir, ipfsspec.CID.String())

	// If the output path already exists, we already have the data, as
	// ipfsClient.Get(...) renames the result path atomically after it has
	// finished downloading the CID.
	ok, err := system.PathExists(outputPath)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	if !ok {
		err = s.ipfsClient.Get(ctx, ipfsspec.CID.String(), outputPath)
		if err != nil {
			return storage.StorageVolume{}, err
		}
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputPath,
		Target: storageSpec.Mount,
	}

	return volume, nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
