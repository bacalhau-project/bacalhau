package ipfs

import (
	"context"

	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type Verifier struct {
	IPFSClient *ipfs_http.IPFSHTTPClient
}

func NewVerifier(cm *system.CleanupManager, ipfsMultiAddress string) (*Verifier, error) {
	api, err := ipfs_http.NewIPFSHTTPClient(ipfsMultiAddress)
	if err != nil {
		return nil, err
	}

	ctx := context.Background() // TODO: instrument
	_, err = api.GetPeerID(ctx)
	if err != nil {
		return nil, err
	}

	v := &Verifier{
		IPFSClient: api,
	}

	url, err := api.GetURL()
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("IPFS verifier initialized with address: %s", url)
	return v, nil
}

func (v *Verifier) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()

	_, err := v.IPFSClient.GetPeerID(ctx)
	return err == nil, err
}

func (v *Verifier) ProcessResultsFolder(ctx context.Context,
	jobID, resultsFolder string) (string, error) {
	ctx, span := newSpan(ctx, "ProcessResultsFolder")
	defer span.End()

	log.Debug().Msgf("Uploading results folder to ipfs: %s %s", jobID, resultsFolder)
	return v.IPFSClient.UploadTar(ctx, resultsFolder)
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "verifier/ipfs", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ verifier.Verifier = (*Verifier)(nil)
