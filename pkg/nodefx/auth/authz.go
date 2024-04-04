package auth

import (
	"github.com/bacalhau-project/bacalhau/pkg/authz"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

func Authorizer(cfg node.NodeConfig) (authz.Authorizer, error) {
	authzPolicy, err := policy.FromPathOrDefault(cfg.AuthConfig.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	signingKey, err := pkgconfig.GetClientPublicKey()
	if err != nil {
		return nil, err
	}
	return authz.NewPolicyAuthorizer(authzPolicy, signingKey, cfg.NodeID), nil
}
