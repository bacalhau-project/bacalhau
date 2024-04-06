package auth

import (
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configfx"
)

var Module = fx.Module("auth",
	fx.Provide(LoadConfig),
	fx.Provide(AuthenticatorsProviders),
	fx.Provide(Authorizer),
)

func LoadConfig(c *configfx.Config) (types.AuthConfig, error) {
	var cfg types.AuthConfig
	if err := c.ForKey(types.Auth, &cfg); err != nil {
		return types.AuthConfig{}, nil
	}

	return cfg, nil
}
