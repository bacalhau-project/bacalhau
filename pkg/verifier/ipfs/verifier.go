package ipfs

import (
	"context"

	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
)

type Verifier struct {
	IPFSClient *ipfs_http.IPFSHttpClient
}

func NewVerifier(cm *system.CleanupManager, ipfsMultiAddress string) (
	*Verifier, error) {

	api, err := ipfs_http.NewIPFSHttpClient(ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	ctx := context.Background() // TODO: instrument
	_, err = api.GetPeerId(ctx)
	if err != nil {
		return nil, err
	}

	verifier := &Verifier{
		IPFSClient: api,
	}

	url, err := api.GetUrl()
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("IPFS verifier initialized with address: %s", url)
	return verifier, nil
}

func (verifier *Verifier) IsInstalled(ctx context.Context) (bool, error) {
	_, err := verifier.IPFSClient.GetPeerId(ctx)
	return err == nil, err
}

func (verifier *Verifier) ProcessResultsFolder(ctx context.Context,
	job *types.Job, resultsFolder string) (string, error) {

	log.Debug().Msgf("Uploading results folder to ipfs: %s %s", job.Id, resultsFolder)
	return verifier.IPFSClient.UploadTar(ctx, resultsFolder)
}

// Compile-time check that Verifier implements the correct interface:
var _ verifier.Verifier = (*Verifier)(nil)
