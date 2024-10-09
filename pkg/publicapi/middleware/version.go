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

type VersionCheckError struct {
	Error         string
	MinVersion    string
	ClientVersion string
	ServerVersion string
}

func VersionNotifyLogger(logger *zerolog.Logger, serverVersion semver.Version) echo.MiddlewareFunc {
	return echomiddelware.RequestLoggerWithConfig(echomiddelware.RequestLoggerConfig{
		// instructs logger to extract given list of headers from request.
		LogHeaders: []string{apimodels.HTTPHeaderBacalhauGitVersion},
		LogValuesFunc: func(c echo.Context, v echomiddelware.RequestLoggerValues) error {
			notification := Notification{
				RequestID:     v.RequestID,
				ClientID:      c.Response().Header().Get(apimodels.HTTPHeaderClientID),
				ServerVersion: serverVersion.String(),
			}

			defer func() {
				if notification.Message != "" {
					logger.WithLevel(zerolog.DebugLevel).
						Str("ClientID", notification.ClientID).
						Str("RequestID", notification.RequestID).
						Str("ClientVersion", notification.ClientVersion).
						Str("ServerVersion", notification.ServerVersion).
						Msg(notification.Message)
				}
			}()

			cVersion := v.Headers[apimodels.HTTPHeaderBacalhauGitVersion]
			if len(cVersion) == 0 {
				// version header is empty, cannot parse it
				notification.Message = "received request from client without version"
				return nil
			}
			if len(cVersion) > 1 {
				// version header contained multiple fields
				notification.Message = fmt.Sprintf("received request from client with multiple versions: %s", cVersion)
				return nil
			}

			// there is a single version header, attempt to parse it.
			clientVersion, err := semver.NewVersion(cVersion[0])
			if err != nil {
				// cannot parse client version, should notify
				notification.Message = fmt.Sprintf("received request with invalid client version: %s", cVersion[0])
				return nil
			}
			// extract parsed client version for comparison
			notification.ClientVersion = clientVersion.String()

			diff := serverVersion.Compare(clientVersion)
			switch diff {
			case 1:
				// client version is less than server version
				notification.Message = "received request from outdated client"
			case -1:
				// server version is less than client version
				notification.Message = "received request from newer client"
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
			if !version.IsVersionExplicit(clientVersionStr) {
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
				return c.JSON(http.StatusForbidden, VersionCheckError{
					Error:         "Client version is outdated. Update your client",
					MinVersion:    minVersion.String(),
					ClientVersion: clientVersion.String(),
					ServerVersion: serverVersion.String(),
				})
			}

			// Client version is acceptable, proceed with the request
			return next(c)
		}
	}
}
