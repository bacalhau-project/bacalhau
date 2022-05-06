package api_copy

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

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
	Ctx        context.Context
	IPFSClient *ipfs_http.IPFSHttpClient
	LocalDir   string
}

func NewIpfsApiCopy(
	ctx context.Context,
	ipfsMultiAddress string,
) (*IpfsApiCopy, error) {
	api, err := ipfs_http.NewIPFSHttpClient(ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}
	dir, err := ioutil.TempDir("", "bacalhau-ipfs")
	if err != nil {
		return nil, err
	}
	storageHandler := &IpfsApiCopy{
		Ctx:        ctx,
		IPFSClient: api,
		LocalDir:   dir,
	}

	log.Debug().Msgf("IPFS API Copy driver created")

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

func (dockerIpfs *IpfsApiCopy) PrepareStorage(storageSpec types.StorageSpec) (*storage.PreparedStorageVolume, error) {

	var statResult struct {
		Hash string
		Type string
	}

	err := dockerIpfs.IPFSClient.Api.Request("files/stat", fmt.Sprintf("/ipfs/%s", storageSpec.Cid)).Exec(dockerIpfs.Ctx, &statResult)

	if err != nil {
		return nil, err
	}

	if statResult.Type == storage.IPFS_TYPE_DIRECTORY || statResult.Type == storage.IPFS_TYPE_FILE {
		return dockerIpfs.copyTarFile(storageSpec)
	} else {
		return nil, fmt.Errorf("unknown ipfs type: %s", statResult.Type)
	}
}

func (dockerIpfs *IpfsApiCopy) CleanupStorage(storageSpec types.StorageSpec, volume *storage.PreparedStorageVolume) error {
	return system.RunCommand("sudo", []string{
		"rm", "-rf", fmt.Sprintf("%s/%s", dockerIpfs.LocalDir, storageSpec.Cid),
	})
}

func (dockerIpfs *IpfsApiCopy) copyTarFile(storageSpec types.StorageSpec) (*storage.PreparedStorageVolume, error) {
	res, err := dockerIpfs.IPFSClient.Api.Request("get", storageSpec.Cid).Send(dockerIpfs.Ctx)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	tarfilePath := fmt.Sprintf("%s/%s.tar", dockerIpfs.LocalDir, storageSpec.Cid)
	log.Debug().Msgf("Writing cid: %s tar file to %s", storageSpec.Cid, tarfilePath)
	outFile, err := os.Create(tarfilePath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, res.Output)
	if err != nil {
		return nil, err
	}
	err = system.RunCommand("tar", []string{
		"-vxf", tarfilePath, "-C", dockerIpfs.LocalDir,
	})
	log.Debug().Msgf("Extracted tar file: %s", tarfilePath)
	err = os.Remove(tarfilePath)
	if err != nil {
		return nil, err
	}
	volume := &storage.PreparedStorageVolume{
		Type:   "bind",
		Source: fmt.Sprintf("%s/%s", dockerIpfs.LocalDir, storageSpec.Cid),
		Target: storageSpec.MountPath,
	}
	return volume, nil
}
