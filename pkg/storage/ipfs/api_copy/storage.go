package api_copy

import (
	"context"
	"fmt"
	"io/ioutil"

	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

// a storage driver runs the downloads content
// from a remote ipfs server and copies it to
// to a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	LocalDir   string
	IPFSClient *ipfs_http.IPFSHttpClient
}

func NewStorageProvider(cm *system.CleanupManager, ipfsMultiAddress string) (
	*StorageProvider, error) {

	api, err := ipfs_http.NewIPFSHttpClient(context.TODO(), ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir("", "bacalhau-ipfs")
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

func (dockerIpfs *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	addresses, err := dockerIpfs.IPFSClient.GetLocalAddrs()
	if err != nil {
		return false, err
	}
	if len(addresses) == 0 {
		return false, fmt.Errorf("No multi addresses loaded from remote ipfs server")
	}
	return true, nil
}

func (dockerIpfs *StorageProvider) HasStorage(ctx context.Context,
	volume types.StorageSpec) (bool, error) {

	return dockerIpfs.IPFSClient.HasCidLocally(volume.Cid)
}

func (dockerIpfs *StorageProvider) PrepareStorage(ctx context.Context,
	storageSpec types.StorageSpec) (*types.StorageVolume, error) {

	var statResult struct {
		Hash string
		Type string
	}

	err := dockerIpfs.IPFSClient.Api.
		Request("files/stat", fmt.Sprintf("/ipfs/%s", storageSpec.Cid)).
		Exec(context.TODO(), &statResult)

	if err != nil {
		return nil, err
	}

	if statResult.Type == storage.IPFS_TYPE_DIRECTORY || statResult.Type == storage.IPFS_TYPE_FILE {
		return dockerIpfs.copyTarFile(storageSpec)
	} else {
		return nil, fmt.Errorf("unknown ipfs type: %s", statResult.Type)
	}
}

func (dockerIpfs *StorageProvider) CleanupStorage(ctx context.Context,
	storageSpec types.StorageSpec, volume *types.StorageVolume) error {

	return system.RunCommand("sudo", []string{
		"rm", "-rf", fmt.Sprintf("%s/%s", dockerIpfs.LocalDir, storageSpec.Cid),
	})
}

func (dockerIpfs *StorageProvider) copyTarFile(storageSpec types.StorageSpec) (*types.StorageVolume, error) {
	err := dockerIpfs.IPFSClient.DownloadTar(dockerIpfs.LocalDir, storageSpec.Cid)
	if err != nil {
		return nil, err
	}
	volume := &types.StorageVolume{
		Type:   "bind",
		Source: fmt.Sprintf("%s/%s", dockerIpfs.LocalDir, storageSpec.Cid),
		Target: storageSpec.Path,
	}
	return volume, nil
}

// Compile time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
