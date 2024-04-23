package publicapi

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"golang.org/x/time/rate"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const EchoTimeoutMessage = "Server Timeout!"

type RouterParams struct {
	fx.In

	Config     types.ServerMiddlewareConfig
	Validator  *CustomValidator
	Logger     *zerolog.Logger
	Authorizer authz.Authorizer

	// TODO allow middleware to be provided, e.g.
	// Middleware []echo.MiddlewareFunc `group:"api_middlewares"`
}

func NewRouter(p RouterParams) (*echo.Echo, error) {
	e := echo.New()

	e.Validator = p.Validator
	e.Debug = p.Config.Debug
	e.Pre(echomiddelware.Rewrite(p.Config.Migrations))
	// set middleware
	logLevel, err := zerolog.ParseLevel(p.Config.LogLevel)
	if err != nil {
		return nil, err
	}

	serverBuildInfo := version.Get()
	serverVersion, err := semver.NewVersion(serverBuildInfo.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to determine server agent version %w", err)
	}
	// base middle after routing
	e.Use(
		echomiddelware.CORS(),
		echomiddelware.Recover(),
		echomiddelware.RequestID(),
		echomiddelware.BodyLimit(p.Config.MaxBytesToReadInBody),
		echomiddelware.RateLimiter(
			echomiddelware.NewRateLimiterMemoryStore(rate.Limit(
				p.Config.ThrottleLimit,
			))),
		echomiddelware.TimeoutWithConfig(
			echomiddelware.TimeoutConfig{
				Timeout:      time.Duration(p.Config.RequestHandlerTimeout),
				ErrorMessage: EchoTimeoutMessage,
				Skipper:      middleware.WebsocketSkipper,
			}),

		middleware.Otel(),
		middleware.Authorize(p.Authorizer),
		// sets headers on the server based on provided config
		middleware.ServerHeader(p.Config.Headers),
		// logs request at appropriate error level based on status code
		middleware.RequestLogger(*p.Logger, logLevel),
		// logs requests made by clients with different versions than the server
		middleware.VersionNotifyLogger(p.Logger, *serverVersion),
	)

	return e, nil
}
