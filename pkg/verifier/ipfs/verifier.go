package ipfs

import (
	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type IPFSVerifier struct {
	cancelContext *system.CancelContext
	IPFSClient    *ipfs_http.IPFSHttpClient
}

func NewIPFSVerifier(
	cancelContext *system.CancelContext,
	ipfsMultiAddress string,
) (*IPFSVerifier, error) {
	api, err := ipfs_http.NewIPFSHttpClient(cancelContext.Ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}
	_, err = api.GetPeerId()
	if err != nil {
		return nil, err
	}
	verifier := &IPFSVerifier{
		cancelContext: cancelContext,
		IPFSClient:    api,
	}
	url, err := api.GetUrl()
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("IPFS verifier initialized with address: %s", url)
	return verifier, nil
}

func (verifier *IPFSVerifier) IsInstalled() (bool, error) {
	return true, nil
}

func (verifier *IPFSVerifier) ProcessResultsFolder(job *types.Job, resultsFolder string) (string, error) {
	log.Debug().Msgf("Uploading results folder to ipfs: %s %s", job.Id, resultsFolder)
	return verifier.IPFSClient.UploadTar(resultsFolder)
}
