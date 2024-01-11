package middleware

import (
	"github.com/Masterminds/semver"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

func VersionNotifyLogger(logger *zerolog.Logger, serverVersion *semver.Version) echo.MiddlewareFunc {
	return echomiddelware.RequestLoggerWithConfig(echomiddelware.RequestLoggerConfig{
		LogHeaders: []string{
			apimodels.HTTPHeaderClientMajorVersion,
			apimodels.HTTPHeaderClientMinorVersion,
			apimodels.HTTPHeaderClientPatchVersion,
			apimodels.HTTPHeaderClientGitVersion,
		},
		LogValuesFunc: func(c echo.Context, v echomiddelware.RequestLoggerValues) error {
			event := logger.WithLevel(zerolog.WarnLevel).
				Str("RequestID", v.RequestID).
				Str("ClientID", c.Response().Header().Get(apimodels.HTTPHeaderClientID))

			notify := false
			cVersion := v.Headers[apimodels.HTTPHeaderClientGitVersion]
			if len(cVersion) == 1 {
				notify = true
				clientVersion, err := semver.NewVersion(cVersion[0])
				if err != nil {
					event.Msgf("received request with invalid client version: %s", cVersion[0])
				} else {
					diff := serverVersion.Compare(clientVersion)
					switch diff {
					case 0:
						// versions are the same, don't notify
						notify = false
					case 1:
						event.
							Str("ServerVersion", serverVersion.String()).
							Str("ClientVersion", clientVersion.String()).
							Msgf("received request from outdated client")
					case -1:
						event.
							Str("ServerVersion", serverVersion.String()).
							Str("ClientVersion", clientVersion.String()).
							Msgf("received request from newer client")
					}
				}
			} else if len(cVersion) == 0 {
				notify = true
				event.Msg("received request from unversioned client")
			} else {
				notify = true
				event.
					Str("ServerVersion", serverVersion.String()).
					Strs("ClientVersions", cVersion).
					Msgf("receieved request from client with multipule versions")
			}

			if notify {
				event.Send()
			}
			return nil
		},
	})
}
