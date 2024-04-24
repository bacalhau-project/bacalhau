package authzfx

import (
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

var Module = fx.Module("authz",
	// fx.Provide(LoadConfig),
	fx.Provide(NewAuthorizer),
)

func LoadConfig(c *config.Config) (types.AuthConfig, error) {
	var cfg types.AuthConfig
	if err := c.ForKey(types.Auth, &cfg); err != nil {
		return types.AuthConfig{}, nil
	}

	return cfg, nil
}

func NewAuthorizer(nodeID types.NodeID, r *repo.FsRepo, cfg types.AuthConfig) (authz.Authorizer, error) {
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
