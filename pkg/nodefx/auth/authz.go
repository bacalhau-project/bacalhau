package auth

import (
	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func Authorizer(nodeID types.NodeID, r *repo.FsRepo, cfg types.AuthConfig) (authz.Authorizer, error) {
	authzPolicy, err := policy.FromPathOrDefault(cfg.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	signingKey, err := r.GetClientPublicKey()
	if err != nil {
		return nil, err
	}
	return authz.NewPolicyAuthorizer(authzPolicy, signingKey, string(nodeID)), nil
}
