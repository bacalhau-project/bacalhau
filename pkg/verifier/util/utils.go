package util

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/deterministic"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
)

func NewStandardVerifiers(
	ctx context.Context,
	cm *system.CleanupManager,
	encrypter verifier.EncrypterFunction,
	decrypter verifier.DecrypterFunction,
) (verifier.VerifierProvider, error) {
	noopVerifier, err := noop.NewNoopVerifier(
		ctx,
		cm,
	)
	if err != nil {
		return nil, err
	}

	deterministicVerifier, err := deterministic.NewDeterministicVerifier(
		ctx,
		cm,
		encrypter,
		decrypter,
	)
	if err != nil {
		return nil, err
	}

	return model.NewMappedProvider(map[model.Verifier]verifier.Verifier{
		model.VerifierNoop:          noopVerifier,
		model.VerifierDeterministic: deterministicVerifier,
	}), nil
}

func NewNoopVerifiers(
	ctx context.Context,
	cm *system.CleanupManager,
	config noop.VerifierConfig,
) (verifier.VerifierProvider, error) {
	noopVerifier, err := noop.NewNoopVerifierWithConfig(ctx, cm, config)
	if err != nil {
		return nil, err
	}
	return model.NewNoopProvider[model.Verifier, verifier.Verifier](noopVerifier), nil
}
