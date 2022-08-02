package ipfs

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type Verifier struct {
	IPFSClient  *ipfs.Client
	JobLoader   job.JobLoader
	StateLoader job.StateLoader
}

func NewVerifier(
	cm *system.CleanupManager,
	ipfsAPIAddr string,
	jobLoader job.JobLoader,
	stateLoader job.StateLoader,
) (*Verifier, error) {
	cl, err := ipfs.NewClient(ipfsAPIAddr)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("IPFS verifier initialized for node: %s", ipfsAPIAddr)
	return &Verifier{
		IPFSClient:  cl,
		JobLoader:   jobLoader,
		StateLoader: stateLoader,
	}, nil
}

func (v *Verifier) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()

	_, err := v.IPFSClient.ID(ctx)
	return err == nil, err
}

func (v *Verifier) ProcessShardResults(
	ctx context.Context,
	jobID string,
	shardIndex int,
	resultsFolder string,
) (string, error) {
	ctx, span := newSpan(ctx, "ProcessResultsFolder")
	defer span.End()

	log.Debug().Msgf("Uploading results folder to ipfs: %s %s", jobID, resultsFolder)
	return v.IPFSClient.Put(ctx, resultsFolder)
}

func (v *Verifier) GetJobResultSet(
	ctx context.Context,
	jobId string,
) ([]storage.StorageSpec, error) {
	return []storage.StorageSpec{}, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "verifier/ipfs", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ verifier.Verifier = (*Verifier)(nil)
