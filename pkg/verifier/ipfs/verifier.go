package ipfs

import (
	"context"

	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type IPFSVerifier struct {
	// Lifecycle context for verifier:
	ctx context.Context

	IPFSClient *ipfs_http.IPFSHttpClient
}

func NewIPFSVerifier(ctx context.Context, ipfsMultiAddress string) (
	*IPFSVerifier, error) {

	api, err := ipfs_http.NewIPFSHttpClient(ctx, ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	_, err = api.GetPeerId()
	if err != nil {
		return nil, err
	}

	verifier := &IPFSVerifier{
		ctx:        ctx,
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
