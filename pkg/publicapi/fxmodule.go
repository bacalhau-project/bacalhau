package publicapi

import (
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var Module = fx.Module("api_server",
	fx.Provide(LoadConfig),
	fx.Provide(NewCustomValidator),
	fx.Provide(NewRouter),
	fx.Provide(NewServer),
)

func LoadConfig(c *config.Config) (types.ServerConfig, types.ServerMiddlewareConfig, error) {
	var svrCfg types.ServerConfig
	if err := c.ForKey(types.NodeServer, &svrCfg); err != nil {
		return types.ServerConfig{}, types.ServerMiddlewareConfig{}, err
	}
	var mdlCfg types.ServerMiddlewareConfig
	if err := c.ForKey(types.NodeServerMiddlewareConfig, &mdlCfg); err != nil {
		return types.ServerConfig{}, types.ServerMiddlewareConfig{}, err
	}

	return svrCfg, mdlCfg, nil
}
