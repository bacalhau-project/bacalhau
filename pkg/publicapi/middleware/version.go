package middleware

import (
	"fmt"
	"net/http"

	"github.com/Masterminds/semver"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
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

func stripPreRelease(version *semver.Version) (*semver.Version, error) {
	newVersion, _ := semver.NewVersion(version.String())
	_, err := newVersion.SetPrerelease("")
	if err != nil {
		return nil, err
	}
	return newVersion, nil
}

func VersionNotifyLogger(logger *zerolog.Logger, serverVersion semver.Version) echo.MiddlewareFunc {
	return echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		// instructs logger to extract given list of headers from request.
		LogHeaders: []string{apimodels.HTTPHeaderBacalhauGitVersion},
		LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
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

			// Strip the pre-release tag
			strippedClientVersion, err := stripPreRelease(clientVersion)
			if err != nil {
				notif.Message = fmt.Sprintf("error stripping pre-release tag from client version: %s", err)
				return nil
			}

			// extract parsed client version for comparison
			notif.ClientVersion = strippedClientVersion.String()

			diff := serverVersion.Compare(strippedClientVersion)
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

			// Strip the pre-release tag
			strippedClientVersion, err := stripPreRelease(clientVersion)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"Error": "Error stripping pre-release tag from client version: " + err.Error(),
				})
			}

			if strippedClientVersion.LessThan(&minVersion) {
				// Client version is less than the minimum required version
				return c.JSON(http.StatusForbidden, VersionCheckError{
					Error:         "Client version is outdated. Update your client",
					MinVersion:    minVersion.String(),
					ClientVersion: strippedClientVersion.String(),
					ServerVersion: serverVersion.String(),
				})
			}

			// Client version is acceptable, proceed with the request
			return next(c)
		}
	}
}
