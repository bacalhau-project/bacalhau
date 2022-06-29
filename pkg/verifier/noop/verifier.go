package noop

import (
	"context"
)

type Verifier struct {
}

func NewVerifier() (*Verifier, error) {
	return &Verifier{}, nil
}

func (v *Verifier) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (v *Verifier) ProcessResultsFolder(ctx context.Context, jobID, resultsFolder string) (string, error) {
	return resultsFolder, nil
}
