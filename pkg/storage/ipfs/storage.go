package ipfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// a storage driver runs the downloads content
// from a remote ipfs server and copies it to
// to a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	LocalDir   string
	IPFSClient ipfs.Client
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
		IPFSClient: cl,
		LocalDir:   dir,
	}

	log.Trace().Msgf("IPFS API Copy driver created with address: %s", cl.APIAddress())
	return storageHandler, nil
}

func (dockerIPFS *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	_, err := dockerIPFS.IPFSClient.ID(ctx)
	return err == nil, err
}

func (dockerIPFS *StorageProvider) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	return dockerIPFS.IPFSClient.HasCID(ctx, volume.CID)
}

// we wrap this in a timeout because if the CID is not present on the network this seems to hang
func (dockerIPFS *StorageProvider) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, config.GetVolumeSizeRequestTimeout(ctx))
	defer cancel()

	return dockerIPFS.IPFSClient.GetCidSize(ctx, volume.CID)
}

func (dockerIPFS *StorageProvider) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	ctx, span := system.GetTracer().Start(ctx, "storage/ipfs/StorageProvider.PrepareStorage")
	defer span.End()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, config.GetDownloadCidRequestTimeout(ctx))
	defer cancel()

	stat, err := dockerIPFS.IPFSClient.Stat(ctx, storageSpec.CID)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to stat %s: %w", storageSpec.CID, err)
	}

	if stat.Type != ipfs.IPLDFile && stat.Type != ipfs.IPLDDirectory {
		return storage.StorageVolume{}, fmt.Errorf("unknown ipld file type for %s: %v", storageSpec.CID, stat.Type)
	}

	var volume storage.StorageVolume
	volume, err = dockerIPFS.getFileFromIPFS(ctx, storageSpec)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to copy %s to volume: %w", storageSpec.Path, err)
	}

	return volume, nil
}

//nolint:lll // Exception to the long rule
func (dockerIPFS *StorageProvider) CleanupStorage(_ context.Context, storageSpec model.StorageSpec, _ storage.StorageVolume) error {
	return os.RemoveAll(filepath.Join(dockerIPFS.LocalDir, storageSpec.CID))
}

func (dockerIPFS *StorageProvider) Upload(ctx context.Context, localPath string) (model.StorageSpec, error) {
	ctx, span := system.GetTracer().Start(ctx, "storage/ipfs/StorageProvider.Upload")
	defer span.End()

	cid, err := dockerIPFS.IPFSClient.Put(ctx, localPath)
	if err != nil {
		return model.StorageSpec{}, err
	}
	return model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           cid,
	}, nil
}

func (dockerIPFS *StorageProvider) Explode(ctx context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	ctx, span := system.GetTracer().Start(ctx, "storage/ipfs/StorageProvider.Explode")
	defer span.End()

	treeNode, err := dockerIPFS.IPFSClient.GetTreeNode(ctx, spec.CID)
	if err != nil {
		return []model.StorageSpec{}, err
	}
	flatNodes, err := ipfs.FlattenTreeNode(ctx, treeNode)
	if err != nil {
		return []model.StorageSpec{}, err
	}
	basePath := strings.TrimPrefix(spec.Path, "/")
	basePath = strings.TrimSuffix(basePath, "/")
	specs := []model.StorageSpec{}
	seenPaths := map[string]bool{}
	for _, node := range flatNodes {
		prepend := basePath
		if prepend != "" {
			prepend = "/" + prepend
		}
		usePath := strings.TrimSuffix(prepend+"/"+strings.Join(node.Path, "/"), "/")
		_, ok := seenPaths[usePath]
		if ok {
			continue
		}
		seenPaths[usePath] = true
		specs = append(specs, model.StorageSpec{
			StorageSource: model.StorageSourceIPFS,
			CID:           node.Cid.String(),
			Path:          usePath,
		})
	}
	return specs, nil
}

func (dockerIPFS *StorageProvider) getFileFromIPFS(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	ctx, span := system.GetTracer().Start(ctx, "storage/ipfs/StorageProvider.copyFile")
	defer span.End()

	outputPath := filepath.Join(dockerIPFS.LocalDir, storageSpec.CID)

	// If the output path already exists, we already have the data, as
	// ipfsClient.Get(...) renames the result path atomically after it has
	// finished downloading the CID.
	ok, err := system.PathExists(outputPath)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	if !ok {
		err = dockerIPFS.IPFSClient.Get(ctx, storageSpec.CID, outputPath)
		if err != nil {
			return storage.StorageVolume{}, err
		}
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputPath,
		Target: storageSpec.Path,
	}

	return volume, nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
