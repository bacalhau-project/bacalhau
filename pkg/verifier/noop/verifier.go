package noop

import (
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type NoopVerifier struct {
}

func NewNoopVerifier() (*NoopVerifier, error) {
	return &NoopVerifier{}, nil
}

func (verifier *NoopVerifier) IsInstalled() (bool, error) {
	return true, nil
}

func (verifier *NoopVerifier) ProcessResultsFolder(job *types.Job, resultsFolder string) (string, error) {
	return resultsFolder, nil
}
