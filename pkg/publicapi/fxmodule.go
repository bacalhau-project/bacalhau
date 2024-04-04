package publicapi

import (
	"go.uber.org/fx"
)

var Module = fx.Module("api_server",
	fx.Provide(NewCustomValidator),
	fx.Provide(NewRouter),
	fx.Provide(NewServer),
)
