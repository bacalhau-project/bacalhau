package util

import (
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/noop"
)

func NewNoopVerifiers(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
) (map[verifier.VerifierType]verifier.Verifier, error) {
	noopVerifier, err := noop.NewNoopVerifier(
		cm,
		resolver,
	)
	if err != nil {
		return nil, err
	}

	return map[verifier.VerifierType]verifier.Verifier{
		verifier.VerifierNoop: noopVerifier,
	}, nil
}
