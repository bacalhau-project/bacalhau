package middleware

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

type Notification struct {
	RequestID     string
	ClientID      string
	ServerVersion string

	ClientVersion string
	Message       string
}

func VersionNotifyLogger(logger *zerolog.Logger, serverVersion semver.Version) echo.MiddlewareFunc {
	return echomiddelware.RequestLoggerWithConfig(echomiddelware.RequestLoggerConfig{
		// instructs logger to extract given list of headers from request.
		LogHeaders: []string{apimodels.HTTPHeaderBacalhauGitVersion},
		LogValuesFunc: func(c echo.Context, v echomiddelware.RequestLoggerValues) error {

			notif := Notification{
				RequestID:     v.RequestID,
				ClientID:      c.Response().Header().Get(apimodels.HTTPHeaderClientID),
				ServerVersion: serverVersion.String(),
			}

			defer func() {
				if notif.Message != "" {
					logger.WithLevel(zerolog.WarnLevel).
						Str("ClientID", notif.ClientID).
						Str("RequestID", notif.RequestID).
						Str("ClientVersion", notif.ClientVersion).
						Str("ServerVersion", notif.ServerVersion).
						Msg(notif.Message)
				}
			}()

			cVersion := v.Headers[apimodels.HTTPHeaderBacalhauGitVersion]
			if len(cVersion) == 0 {
				// version header is empty, cannot parse it
				notif.Message = "received request from client without version"
				return nil
			}
			if len(cVersion) > 1 {
				// version header contained multiple fields
				notif.Message = fmt.Sprintf("received request from client with multiple versions: %s", cVersion)
				return nil
			}

			// there is a single version header, attempt to parse it.
			clientVersion, err := semver.NewVersion(cVersion[0])
			if err != nil {
				// cannot parse client version, should notify
				notif.Message = fmt.Sprintf("received request with invalid client version: %s", cVersion[0])
				return nil
			}
			// extract parsed client version for comparison
			notif.ClientVersion = clientVersion.String()

			diff := serverVersion.Compare(clientVersion)
			switch diff {
			case 1:
				// client version is less than server version
				notif.Message = "received request from outdated client"
			case -1:
				// server version is less than client version
				notif.Message = "received request from newer client"
			case 0:
				// versions are the same, don't notify
			}

			return nil
		},
	})
}
