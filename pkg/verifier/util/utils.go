package util

import (
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/deterministic"
	"github.com/filecoin-project/bacalhau/pkg/verifier/noop"
)

func NewStandardVerifiers(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
	encrypter verifier.EncrypterFunction,
	decrypter verifier.DecrypterFunction,
) (map[model.VerifierType]verifier.Verifier, error) {
	noopVerifier, err := noop.NewNoopVerifier(
		cm,
		resolver,
	)
	if err != nil {
		return nil, err
	}

	deterministicVerifier, err := deterministic.NewDeterministicVerifier(
		cm,
		resolver,
		encrypter,
		decrypter,
	)
	if err != nil {
		return nil, err
	}

	return map[model.VerifierType]verifier.Verifier{
		model.VerifierNoop:          noopVerifier,
		model.VerifierDeterministic: deterministicVerifier,
	}, nil
}

func NewNoopVerifiers(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
) (map[model.VerifierType]verifier.Verifier, error) {
	noopVerifier, err := noop.NewNoopVerifier(
		cm,
		resolver,
	)
	if err != nil {
		return nil, err
	}

	return map[model.VerifierType]verifier.Verifier{
		model.VerifierNoop:          noopVerifier,
		model.VerifierDeterministic: noopVerifier,
	}, nil
}
