package v2

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/ask"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
)

func SetupAuthenticators(privKey *rsa.PrivateKey, cfg v2.Bacalhau) (authn.Provider, error) {
	var allErr error
	authns := make(map[string]authn.Authenticator, len(cfg.Server.Auth.Methods))
	for name, authnConfig := range cfg.Server.Auth.Methods {
		switch authnConfig.Type {
		case authn.MethodTypeChallenge:
			methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = challenge.NewAuthenticator(
				methodPolicy,
				challenge.NewStringMarshaller(cfg.Name),
				privKey,
				cfg.Name,
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
				cfg.Name,
			)
		default:
			allErr = errors.Join(allErr, fmt.Errorf("unknown authentication type: %q", authnConfig.Type))
		}
	}

	return provider.NewMappedProvider(authns), allErr
}
