package auth

import (
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/ask"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func AuthenticatorsProviders(nodeID types.NodeID, r *repo.FsRepo, cfg types.AuthConfig) (authn.Provider, error) {
	privKey, err := r.GetClientPrivateKey()
	if err != nil {
		return nil, err
	}

	var allErr error
	authns := make(map[string]authn.Authenticator, len(cfg.Methods))
	for name, authnConfig := range cfg.Methods {
		switch authnConfig.Type {
		case authn.MethodTypeChallenge:
			methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = challenge.NewAuthenticator(
				methodPolicy,
				challenge.NewStringMarshaller(string(nodeID)),
				privKey,
				string(nodeID),
			)
		case authn.MethodTypeAsk:
			methodPolicy, err := policy.FromPath(authnConfig.PolicyPath)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = ask.NewAuthenticator(
				methodPolicy,
				privKey,
				string(nodeID),
			)
		default:
			allErr = errors.Join(allErr, fmt.Errorf("unknown authentication type: %q", authnConfig.Type))
		}
	}

	return provider.NewMappedProvider(authns), allErr
}
