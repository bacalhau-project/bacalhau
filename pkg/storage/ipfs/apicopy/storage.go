package apicopy

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	ipfsHTTP "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// a storage driver runs the downloads content
// from a remote ipfs server and copies it to
// to a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	LocalDir   string
	IPFSClient *ipfsHTTP.IPFSHTTPClient
}

func NewStorageProvider(cm *system.CleanupManager, ipfsMultiAddress string) (*StorageProvider, error) {
	api, err := ipfsHTTP.NewIPFSHTTPClient(ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	// TODO: consolidate the various config inputs into one package otherwise they are scattered across the codebase
	dir, err := ioutil.TempDir(os.Getenv("BACALHAU_STORAGE_PATH"), "bacalhau-ipfs")
	if err != nil {
		return nil, err
	}

	storageHandler := &StorageProvider{
		IPFSClient: api,
		LocalDir:   dir,
	}

	log.Debug().Msgf("IPFS API Copy driver created with address: %s", ipfsMultiAddress)

	return storageHandler, nil
}

func (dockerIPFS *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()

	addresses, err := dockerIPFS.IPFSClient.GetLocalAddrs(ctx)
	if err != nil {
		return false, err
	}
	if len(addresses) == 0 {
		return false, fmt.Errorf("no multi addresses loaded from remote ipfs server")
	}
	return true, nil
}

func (dockerIPFS *StorageProvider) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()

	return dockerIPFS.IPFSClient.HasCidLocally(ctx, volume.Cid)
}

func (sp *StorageProvider) GetVolumeSize(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
	ctx, span := newSpan(ctx, "GetVolumeResourceUsage")
	defer span.End()
	return 0, nil
}

func (dockerIPFS *StorageProvider) PrepareStorage(ctx context.Context, storageSpec storage.StorageSpec) (*storage.StorageVolume, error) {
	ctx, span := newSpan(ctx, "PrepareStorage")
	defer span.End()

	var statResult struct {
		Hash string
		Type string
	}

	err := dockerIPFS.IPFSClient.API.
		Request("files/stat", fmt.Sprintf("/ipfs/%s", storageSpec.Cid)).
		Exec(ctx, &statResult)

	if err != nil {
		return nil, err
	}

	if statResult.Type == storage.IPFSTypeDirectory || statResult.Type == storage.IPFSTypeFile {
		return dockerIPFS.copyTarFile(ctx, storageSpec)
	} else {
		return nil, fmt.Errorf("unknown ipfs type: %s", statResult.Type)
	}
}

// nolint:lll // Exception to the long rule
func (dockerIPFS *StorageProvider) CleanupStorage(ctx context.Context, storageSpec storage.StorageSpec, volume *storage.StorageVolume) error {
	return system.RunCommand("sudo", []string{
		"rm", "-rf", fmt.Sprintf("%s/%s", dockerIPFS.LocalDir, storageSpec.Cid),
	})
}

func (dockerIPFS *StorageProvider) copyTarFile(ctx context.Context, storageSpec storage.StorageSpec) (*storage.StorageVolume, error) {
	err := dockerIPFS.IPFSClient.DownloadTar(ctx,
		dockerIPFS.LocalDir, storageSpec.Cid)
	if err != nil {
		return nil, err
	}

	volume := &storage.StorageVolume{
		Type:   "bind",
		Source: fmt.Sprintf("%s/%s", dockerIPFS.LocalDir, storageSpec.Cid),
		Target: storageSpec.Path,
	}

	return volume, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "storage/ipfs/api_copy", apiName)
}

// Compile time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
