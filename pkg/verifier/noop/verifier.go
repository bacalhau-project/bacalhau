package noop

import (
	"context"
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
	jobID, resultsFolder string) (string, error) {

	return resultsFolder, nil
}
