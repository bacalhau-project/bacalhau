package middleware

import (
	"fmt"
	"net/http"
	"regexp"

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

// parseVersion parses the version string and strips the pre-release tag.
func parseVersion(versionStr string) (*semver.Version, error) {
	// Strip build metadata from the version string
	re := regexp.MustCompile(`^\d+\.\d+\.\d+(-\d+)?(\+\d+)?$`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 1 {
		return nil, fmt.Errorf("invalid version format")
	}
	return semver.NewVersion(matches[0])
}

func VersionNotifyLogger(logger *zerolog.Logger, serverVersion semver.Version) echo.MiddlewareFunc {
	return echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
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
				notif.Message = "received request from client without version"
				return nil
			}
			if len(cVersion) > 1 {
				notif.Message = fmt.Sprintf("received request from client with multiple versions: %s", cVersion)
				return nil
			}

			clientVersion, err := parseVersion(cVersion[0])
			if err != nil {
				notif.Message = fmt.Sprintf("received request with invalid client version: %s", cVersion[0])
				logger.Error().Msgf("Failed to parse client version: %s", err)
				return nil
			}
			notif.ClientVersion = clientVersion.String()

			diff := serverVersion.Compare(clientVersion)
			switch diff {
			case 1:
				notif.Message = "received request from outdated client"
			case -1:
				notif.Message = "received request from newer client"
			case 0:
			}

			return nil
		},
	})
}

func VersionCheckMiddleware(serverVersion, minVersion semver.Version) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientVersionStr := c.Request().Header.Get(apimodels.HTTPHeaderBacalhauGitVersion)
			if clientVersionStr == "" ||
				clientVersionStr == version.DevelopmentGitVersion ||
				clientVersionStr == version.UnknownGitVersion {
				return next(c)
			}

			clientVersion, err := parseVersion(clientVersionStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"Error": "Invalid client version " + clientVersionStr,
				})
			}

			if clientVersion.LessThan(&minVersion) {
				return c.JSON(http.StatusForbidden, VersionCheckError{
					Error:         "Client version is outdated. Update your client",
					MinVersion:    minVersion.String(),
					ClientVersion: clientVersion.String(),
					ServerVersion: serverVersion.String(),
				})
			}

			return next(c)
		}
	}
}
