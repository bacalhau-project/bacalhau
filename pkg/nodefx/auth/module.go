package auth

import (
	"go.uber.org/fx"
)

var Module = fx.Module("auth",
	fx.Provide(AuthenticatorsProviders),
	fx.Provide(Authorizer),
)
