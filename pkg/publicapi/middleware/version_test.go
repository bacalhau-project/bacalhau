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
)

type VersionNotifyTestSuite struct {
	suite.Suite
	logger zerolog.Logger
	buf    *bytes.Buffer
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
				notif := suite.parseMessage(suite.buf.String())
				suite.Contains(notif.Message, tc.expectedMessage)
				suite.Equal(tc.expectedClientVersion, notif.ClientVersion)
			}
		})
	}
}

func (suite *VersionNotifyTestSuite) parseMessage(msg string) Notification {
	var out Notification
	suite.Require().NoError(json.Unmarshal([]byte(msg), &out))
	return out
}
