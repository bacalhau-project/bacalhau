//go:build unit || !integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TimeoutMiddlewareTestSuite struct {
	suite.Suite
	longRunningHandler  http.Handler
	shortRunningHandler http.Handler
}

func (suite *TimeoutMiddlewareTestSuite) SetupTest() {
	suite.longRunningHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("Success"))
	})

	suite.shortRunningHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Success"))
	})
}

func (suite *TimeoutMiddlewareTestSuite) TestTimeoutOccurs() {
	middleware := Timeout(100 * time.Millisecond)
	handler := middleware(suite.longRunningHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rr, req)

	suite.Contains(rr.Body.String(), DefaultTimeoutMessage)
	suite.Equal(http.StatusServiceUnavailable, rr.Code)
}

func (suite *TimeoutMiddlewareTestSuite) TestTimeoutDoesNotOccur() {
	middleware := Timeout(300 * time.Millisecond)
	handler := middleware(suite.longRunningHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rr, req)

	suite.Equal("Success", rr.Body.String())
	suite.Equal(http.StatusOK, rr.Code)
}

func (suite *TimeoutMiddlewareTestSuite) TestSkippedPaths() {
	middleware := TimeoutWithConfig(TimeoutConfig{
		Timeout:      200 * time.Millisecond,
		SkippedPaths: []string{"/skip"},
	})
	handler := middleware(suite.longRunningHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/skip", nil)

	handler.ServeHTTP(rr, req)

	suite.Equal("Success", rr.Body.String())
	suite.Equal(http.StatusOK, rr.Code)
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(TimeoutMiddlewareTestSuite))
}
