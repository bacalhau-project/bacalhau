package util

import (
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/noop"
)

func NewNoopVerifiers(
	cm *system.CleanupManager,
	jobLoader job.JobLoader,
	stateLoader job.StateLoader,
) (map[verifier.VerifierType]verifier.Verifier, error) {
	noopVerifier, err := noop.NewNoopVerifier(
		cm,
		jobLoader,
		stateLoader,
	)
	if err != nil {
		return nil, err
	}

	return map[verifier.VerifierType]verifier.Verifier{
		verifier.VerifierNoop: noopVerifier,
	}, nil
}
