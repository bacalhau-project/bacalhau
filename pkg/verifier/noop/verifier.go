package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
)

type Verifier struct {
}

func NewVerifier() (*Verifier, error) {
	return &Verifier{}, nil
}

func (v *Verifier) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (v *Verifier) ProcessShardResults(
	ctx context.Context,
	jobID string,
	shardIndex int,
	resultsFolder string,
) (string, error) {
	return resultsFolder, nil
}

func (v *Verifier) GetJobResultSet(
	ctx context.Context,
	jobID string,
) ([]storage.StorageSpec, error) {
	return []storage.StorageSpec{}, nil
}

// Compile-time check that Verifier implements the correct interface:
var _ verifier.Verifier = (*Verifier)(nil)
