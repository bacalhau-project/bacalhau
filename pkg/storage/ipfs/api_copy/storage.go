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

type IpfsApiCopy struct {
	// Lifecycle context for storage driver:
	ctx context.Context

	LocalDir   string
	IPFSClient *ipfs_http.IPFSHttpClient
}

func NewIpfsApiCopy(ctx context.Context, ipfsMultiAddress string) (
	*IpfsApiCopy, error) {

	api, err := ipfs_http.NewIPFSHttpClient(ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	dir, err := ioutil.TempDir("", "bacalhau-ipfs")
	if err != nil {
		return nil, err
	}

	storageHandler := &IpfsApiCopy{
		ctx:        ctx,
		IPFSClient: api,
		LocalDir:   dir,
	}

	log.Debug().Msgf("IPFS API Copy driver created with address: %s", ipfsMultiAddress)

	return storageHandler, nil
}

func (dockerIpfs *IpfsApiCopy) IsInstalled() (bool, error) {
	addresses, err := dockerIpfs.IPFSClient.GetLocalAddrs()
	if err != nil {
		return false, err
	}
	if len(addresses) == 0 {
		return false, fmt.Errorf("No multi addresses loaded from remote ipfs server")
	}
	return true, nil
}

func (dockerIpfs *IpfsApiCopy) HasStorage(volume types.StorageSpec) (bool, error) {
	return dockerIpfs.IPFSClient.HasCidLocally(volume.Cid)
}

func (dockerIpfs *IpfsApiCopy) PrepareStorage(storageSpec types.StorageSpec) (*types.StorageVolume, error) {

	var statResult struct {
		Hash string
		Type string
	}

	err := dockerIpfs.IPFSClient.Api.
		Request("files/stat", fmt.Sprintf("/ipfs/%s", storageSpec.Cid)).
		Exec(dockerIpfs.ctx, &statResult)

	if err != nil {
		return nil, err
	}

	if statResult.Type == storage.IPFS_TYPE_DIRECTORY || statResult.Type == storage.IPFS_TYPE_FILE {
		return dockerIpfs.copyTarFile(storageSpec)
	} else {
		return nil, fmt.Errorf("unknown ipfs type: %s", statResult.Type)
	}
}

func (dockerIpfs *IpfsApiCopy) CleanupStorage(storageSpec types.StorageSpec, volume *types.StorageVolume) error {
	return system.RunCommand("sudo", []string{
		"rm", "-rf", fmt.Sprintf("%s/%s", dockerIpfs.LocalDir, storageSpec.Cid),
	})
}

func (dockerIpfs *IpfsApiCopy) copyTarFile(storageSpec types.StorageSpec) (*types.StorageVolume, error) {
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
