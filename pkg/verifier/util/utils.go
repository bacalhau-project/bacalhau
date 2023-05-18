package util

import (
	"context"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/deterministic"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/external"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
	"go.uber.org/multierr"
)

func NewStandardVerifiers(
	ctx context.Context,
	cm *system.CleanupManager,
	publishers publisher.PublisherProvider,
	externalWebhook *url.URL,
	encrypter verifier.EncrypterFunction,
	decrypter verifier.DecrypterFunction,
) (provider verifier.VerifierProvider, rerr error) {
	verifiers := model.NewMappedProvider(map[model.Verifier]verifier.Verifier{})

	noopVerifier, err := noop.NewNoopVerifier(ctx, cm)
	rerr = multierr.Append(rerr, err)
	if err == nil {
		verifiers.Add(model.VerifierNoop, noopVerifier)
	}

	deterministicVerifier, err := deterministic.NewDeterministicVerifier(ctx, cm, encrypter, decrypter)
	rerr = multierr.Append(rerr, err)
	if err == nil {
		verifiers.Add(model.VerifierDeterministic, deterministicVerifier)
	}

	if externalWebhook != nil {
		externalVerifier, err := external.NewExternalVerifier(publishers, externalWebhook)
		rerr = multierr.Append(rerr, err)
		if err == nil {
			verifiers.Add(model.VerifierExternal, externalVerifier)
		}
	}

	return verifiers, rerr
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
