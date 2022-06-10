package ipfs

import (
	"context"

	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type IPFSVerifier struct {
	IPFSClient *ipfs_http.IPFSHttpClient
}

func NewIPFSVerifier(cm *system.CleanupManager, ipfsMultiAddress string) (
	*IPFSVerifier, error) {

	api, err := ipfs_http.NewIPFSHttpClient(context.TODO(), ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	_, err = api.GetPeerId()
	if err != nil {
		return nil, err
	}

	verifier := &IPFSVerifier{
		IPFSClient: api,
	}

	url, err := api.GetUrl()
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("IPFS verifier initialized with address: %s", url)
	return verifier, nil
}

func (verifier *IPFSVerifier) IsInstalled() (bool, error) {
	_, err := verifier.IPFSClient.GetPeerId()
	return err == nil, err
}

func (verifier *IPFSVerifier) ProcessResultsFolder(job *types.Job, resultsFolder string) (string, error) {
	log.Debug().Msgf("Uploading results folder to ipfs: %s %s", job.Id, resultsFolder)
	return verifier.IPFSClient.UploadTar(resultsFolder)
}
