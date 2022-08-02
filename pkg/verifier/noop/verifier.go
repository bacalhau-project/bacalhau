package noop

import (
	"context"

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

func (v *Verifier) ProcessShardResultsFolder(
	ctx context.Context,
	jobID string,
	shardIndex int,
	resultsFolder string,
) (string, error) {
	return resultsFolder, nil
}

// Compile-time check that Verifier implements the correct interface:
var _ verifier.Verifier = (*Verifier)(nil)
