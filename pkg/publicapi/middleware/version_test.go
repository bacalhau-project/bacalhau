//go:build unit || !integration

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type VersionNotifyTestSuite struct {
	suite.Suite
	logger        zerolog.Logger
	buf           *bytes.Buffer
	originalLevel zerolog.Level
}

func (suite *VersionNotifyTestSuite) SetupSuite() {
	// Store original level
	suite.originalLevel = zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
}

func (suite *VersionNotifyTestSuite) TearDownSuite() {
	// Restore original level
	zerolog.SetGlobalLevel(suite.originalLevel)
}

func (suite *VersionNotifyTestSuite) SetupTest() {
	suite.buf = &bytes.Buffer{}
	suite.logger = zerolog.New(suite.buf)
}

func TestLogVersionNotifyTestSute(t *testing.T) {
	suite.Run(t, new(VersionNotifyTestSuite))
}

func (suite *VersionNotifyTestSuite) TestLogVersionNotify() {
	for _, tc := range []struct {
		name                  string
		clientVersion         []string
		serverVersion         *semver.Version
		expectedMessage       string
		expectedClientVersion string
	}{
		{
			name:            "same version: no notification",
			serverVersion:   semver.MustParse("v1.2.3"),
			clientVersion:   []string{semver.MustParse("v1.2.3").String()},
			expectedMessage: "",
		},
		{
			name:            "no version: notify",
			serverVersion:   semver.MustParse("v1.2.3"),
			clientVersion:   []string{},
			expectedMessage: "received request from client without version",
		},
		{
			name:            "multiple versions: notify",
			serverVersion:   semver.MustParse("v1.2.3"),
			clientVersion:   []string{"v1.0.0", "v1.1.0"},
			expectedMessage: "received request from client with multiple versions",
		},
		{
			name:                  "different major: client outdated notify",
			serverVersion:         semver.MustParse("v1.2.3"),
			clientVersion:         []string{semver.MustParse("v1.2.2").String()},
			expectedMessage:       "received request from outdated client",
			expectedClientVersion: semver.MustParse("v1.2.2").String(),
		},
		{
			name:                  "different major: client newer notify",
			serverVersion:         semver.MustParse("v1.2.3"),
			clientVersion:         []string{semver.MustParse("v1.2.4").String()},
			expectedMessage:       "received request from newer client",
			expectedClientVersion: semver.MustParse("v1.2.4").String(),
		},
		{
			name:            "invalid client version: notify",
			serverVersion:   semver.MustParse("v1.2.3"),
			clientVersion:   []string{"invalid version string"},
			expectedMessage: "received request with invalid client version:",
		},
	} {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			suite.buf.Reset()

			router := echo.New()
			router.Use(VersionNotifyLogger(&suite.logger, *tc.serverVersion))
			router.GET("/test", func(e echo.Context) error {
				return nil
			})

			req, _ := http.NewRequestWithContext(ctx, "GET", "/test", nil)
			for _, h := range tc.clientVersion {
				req.Header.Add(apimodels.HTTPHeaderBacalhauGitVersion, h)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if suite.buf.Len() == 0 {
				suite.Equalf("", tc.expectedMessage, "unexpected notification")
			} else {
				notification := suite.parseMessage(suite.buf.String())
				suite.Contains(notification.Message, tc.expectedMessage)
				suite.Equal(tc.expectedClientVersion, notification.ClientVersion)
			}
		})
	}
}

func (suite *VersionNotifyTestSuite) parseMessage(msg string) Notification {
	var out Notification
	suite.Require().NoError(json.Unmarshal([]byte(msg), &out))
	return out
}

type VersionCheckTestSuite struct {
	suite.Suite
}

func TestVersionCheckTestSuite(t *testing.T) {
	suite.Run(t, new(VersionCheckTestSuite))
}

func (suite *VersionCheckTestSuite) TestVersionCheckMiddleware() {
	for _, tc := range []struct {
		name             string
		clientVersion    string
		minVersion       *semver.Version
		serverVersion    *semver.Version
		expectedStatus   int
		expectedResponse string
	}{
		{
			name:           "no version header",
			clientVersion:  "",
			minVersion:     semver.MustParse("v1.2.3"),
			serverVersion:  semver.MustParse("v1.2.3"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "development version",
			clientVersion:  version.DevelopmentGitVersion,
			minVersion:     semver.MustParse("v1.2.3"),
			serverVersion:  semver.MustParse("v1.2.3"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unknown version",
			clientVersion:  version.UnknownGitVersion,
			minVersion:     semver.MustParse("v1.2.3"),
			serverVersion:  semver.MustParse("v1.2.3"),
			expectedStatus: http.StatusOK,
		},
		{
			name:             "invalid version header",
			clientVersion:    "invalid_version",
			minVersion:       semver.MustParse("v1.2.3"),
			serverVersion:    semver.MustParse("v1.2.3"),
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: "Invalid client version invalid_version",
		},
		{
			name:             "client version less than minimum",
			clientVersion:    "v1.0.0",
			minVersion:       semver.MustParse("v1.2.3"),
			serverVersion:    semver.MustParse("v1.2.3"),
			expectedStatus:   http.StatusForbidden,
			expectedResponse: "Client version is outdated. Update your client",
		},
		{
			name:           "client version meets minimum",
			clientVersion:  "v1.2.3",
			minVersion:     semver.MustParse("v1.2.3"),
			serverVersion:  semver.MustParse("v1.2.3"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "client version greater than minimum",
			clientVersion:  "v1.3.0",
			minVersion:     semver.MustParse("v1.2.3"),
			serverVersion:  semver.MustParse("v1.2.3"),
			expectedStatus: http.StatusOK,
		},
	} {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			router := echo.New()
			router.Use(VersionCheckMiddleware(*tc.serverVersion, *tc.minVersion))
			router.GET("/test", func(e echo.Context) error {
				return e.String(http.StatusOK, "OK")
			})

			req, _ := http.NewRequestWithContext(ctx, "GET", "/test", nil)
			if tc.clientVersion != "" {
				req.Header.Add(apimodels.HTTPHeaderBacalhauGitVersion, tc.clientVersion)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			suite.Equal(tc.expectedStatus, rr.Code)
			if tc.expectedStatus == http.StatusForbidden {
				var response VersionCheckError
				suite.NoError(json.Unmarshal(rr.Body.Bytes(), &response))
				suite.Equal(tc.expectedResponse, response.Error)
				suite.Equal(tc.serverVersion.String(), response.ServerVersion)
				suite.Equal(tc.minVersion.String(), response.MinVersion)
				suite.Equal(tc.clientVersion, "v"+response.ClientVersion)
			}
		})
	}
}
