//go:build unit || !integration

package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ZeroLogFormatterTestSuite struct {
	suite.Suite
	logger zerolog.Logger
	buf    *bytes.Buffer
}

func (suite *ZeroLogFormatterTestSuite) SetupTest() {
	suite.buf = &bytes.Buffer{}
	suite.logger = zerolog.New(suite.buf)
}

// TestLog200 tests that a 200 status code is logged at info level.
func (suite *ZeroLogFormatterTestSuite) TestLogLevels() {
	for _, tc := range []struct {
		name           string
		logLevel       zerolog.Level
		returnedStatus int
		expectedLevel  string
	}{
		{
			name:           "info for 200",
			logLevel:       zerolog.InfoLevel,
			returnedStatus: http.StatusOK,
			expectedLevel:  "info",
		},
		{
			name:           "warn for 400",
			logLevel:       zerolog.InfoLevel,
			returnedStatus: http.StatusBadRequest,
			expectedLevel:  "warn",
		},
		{
			name:           "error for 500",
			logLevel:       zerolog.InfoLevel,
			returnedStatus: http.StatusInternalServerError,
			expectedLevel:  "error",
		},
		{
			name:           "logLevel:debug return debug for 200",
			logLevel:       zerolog.DebugLevel,
			returnedStatus: http.StatusOK,
			expectedLevel:  "debug",
		},
		{
			name:           "logLevel:debug return warn for 400",
			logLevel:       zerolog.DebugLevel,
			returnedStatus: http.StatusBadRequest,
			expectedLevel:  "warn",
		},
		{
			name:           "logLevel:debug return debug for 500",
			logLevel:       zerolog.DebugLevel,
			returnedStatus: http.StatusInternalServerError,
			expectedLevel:  "error",
		},
		{
			name:           "logLevel:error return debug for 200",
			logLevel:       zerolog.ErrorLevel,
			returnedStatus: http.StatusOK,
			expectedLevel:  "error",
		},
		{
			name:           "logLevel:fatal return debug for 500",
			logLevel:       zerolog.FatalLevel,
			returnedStatus: http.StatusInternalServerError,
			expectedLevel:  "fatal",
		},
	} {
		suite.Run(tc.name, func() {
			formatter := NewZeroLogFormatter(
				WithLogger(suite.logger),
				WithLogLevel(tc.logLevel),
			)

			suite.buf.Reset()
			router := chi.NewRouter()
			router.Use(chimiddleware.RequestLogger(formatter))
			router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.returnedStatus)
			})
			rr := httptest.NewRecorder()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			router.ServeHTTP(rr, req)
			suite.Contains(suite.buf.String(), tc.expectedLevel)
			suite.Contains(suite.buf.String(), fmt.Sprintf(`"StatusCode":%d`, tc.returnedStatus))
		})
	}
}

func (suite *ZeroLogFormatterTestSuite) TestPanic() {
	formatter := NewZeroLogFormatter(WithLogger(suite.logger))
	entry := formatter.NewLogEntry(httptest.NewRequest(http.MethodGet, "/", nil))
	entry.Panic("test panic", []byte("stacktrace"))

	assert.Contains(suite.T(), suite.buf.String(), `test panic`)
}

func TestZeroLogFormatterTestSuite(t *testing.T) {
	suite.Run(t, new(ZeroLogFormatterTestSuite))
}
