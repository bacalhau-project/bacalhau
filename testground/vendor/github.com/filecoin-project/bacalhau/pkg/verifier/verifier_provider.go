package verifier

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// A simple verifier repo that selects a verifier based on the job's verifier type.
type MappedVerifierProvider struct {
	verifiers               map[model.Verifier]Verifier
	verifiersInstalledCache map[model.Verifier]bool
}

func NewMappedVerifierProvider(verifiers map[model.Verifier]Verifier) *MappedVerifierProvider {
	return &MappedVerifierProvider{
		verifiers:               verifiers,
		verifiersInstalledCache: map[model.Verifier]bool{},
	}
}

func (p *MappedVerifierProvider) GetVerifier(ctx context.Context, verifierType model.Verifier) (Verifier, error) {
	verifier, ok := p.verifiers[verifierType]
	if !ok {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", verifierType)
	}

	// cache it being installed so we're not hammering it
	// TODO: we should evict the cache in case an installed verifier gets uninstalled, or vice versa
	installed, ok := p.verifiersInstalledCache[verifierType]
	var err error
	if !ok {
		installed, err = verifier.IsInstalled(ctx)
		if err != nil {
			return nil, err
		}
		p.verifiersInstalledCache[verifierType] = installed
	}

	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", verifierType)
	}

	return verifier, nil
}
