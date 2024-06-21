package middleware

import (
	"fmt"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/version"
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

// VersionCheckMiddleware returns a middleware that checks if the client version is at least minVersion.
func VersionCheckMiddleware(serverVersion, minVersion semver.Version) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientVersionStr := c.Request().Header.Get(apimodels.HTTPHeaderBacalhauGitVersion)
			if clientVersionStr == "" ||
				clientVersionStr == version.DevelopmentGitVersion ||
				clientVersionStr == version.UnknownGitVersion {
				// allow the request to pass through
				return next(c)
			}

			clientVersion, err := semver.NewVersion(clientVersionStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"Error": "Invalid client version " + clientVersionStr,
				})
			}

			if clientVersion.LessThan(&minVersion) {
				// Client version is less than the minimum required version
				return c.JSON(http.StatusForbidden, map[string]string{
					"Error":          "Client version is outdated. Update your client",
					"ServerVersion":  serverVersion.String(),
					"MinimumVersion": minVersion.String(),
				})
			}

			// Client version is acceptable, proceed with the request
			return next(c)
		}
	}
}
