package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
)

type Verifier struct {
}

func NewVerifier() (*Verifier, error) {
	return &Verifier{}, nil
}

func (verifier *Verifier) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (verifier *Verifier) ProcessResultsFolder(ctx context.Context,
	job *executor.Job, resultsFolder string) (string, error) {

	return resultsFolder, nil
}
