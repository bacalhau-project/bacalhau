package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

type CustomHTTPErrorHandlerTestSuite struct {
	suite.Suite
	echo *echo.Echo
}

func (suite *CustomHTTPErrorHandlerTestSuite) SetupTest() {
	suite.echo = echo.New()
	suite.echo.HTTPErrorHandler = CustomHTTPErrorHandler
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestBaseError() {
	tests := []struct {
		name string
		err  bacerrors.Error
		want apimodels.APIError
	}{
		{
			name: "basic error",
			err: bacerrors.New("test base error").
				WithHTTPStatusCode(http.StatusBadRequest).
				WithCode("TEST_ERROR").
				WithComponent("TEST_COMPONENT"),
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "test base error",
				Code:           "TEST_ERROR",
				Component:      "TEST_COMPONENT",
			},
		},
		{
			name: "error with hint",
			err: bacerrors.New("test error with hint").
				WithHTTPStatusCode(http.StatusBadRequest).
				WithCode("TEST_ERROR").
				WithComponent("TEST_COMPONENT").
				WithHint("test hint"),
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "test error with hint",
				Code:           "TEST_ERROR",
				Component:      "TEST_COMPONENT",
				Hint:           "test hint",
			},
		},
		{
			name: "error with details",
			err: bacerrors.New("test error with details").
				WithHTTPStatusCode(http.StatusBadRequest).
				WithCode("TEST_ERROR").
				WithComponent("TEST_COMPONENT").
				WithDetails(map[string]string{"key": "value"}),
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "test error with details",
				Code:           "TEST_ERROR",
				Component:      "TEST_COMPONENT",
				Details:        map[string]string{"key": "value"},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := suite.echo.NewContext(req, rec)

			CustomHTTPErrorHandler(tt.err, c)

			suite.Equal(tt.want.HTTPStatusCode, rec.Result().StatusCode)

			var apiError apimodels.APIError
			suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

			suite.Equal(tt.want.Message, apiError.Message)
			suite.Equal(tt.want.Code, apiError.Code)
			suite.Equal(tt.want.Component, apiError.Component)
			suite.Equal(tt.want.Hint, apiError.Hint)
			if tt.want.Details != nil {
				suite.Equal(tt.want.Details, apiError.Details)
			}
		})
	}
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestEchoHTTPError() {
	tests := []struct {
		name      string
		err       *echo.HTTPError
		debugMode bool
		want      apimodels.APIError
	}{
		{
			name:      "basic echo error",
			err:       echo.NewHTTPError(http.StatusUnauthorized, "unauthorized access"),
			debugMode: false,
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusUnauthorized,
				Message:        "unauthorized access",
				Code:           string(bacerrors.InternalError),
				Component:      "APIServer",
			},
		},
		{
			name: "echo error with internal error in debug mode",
			err: &echo.HTTPError{
				Code:     http.StatusBadRequest,
				Message:  "bad request",
				Internal: errors.New("internal error details"),
			},
			debugMode: true,
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "bad request. internal error details",
				Code:           string(bacerrors.InternalError),
				Component:      "APIServer",
			},
		},
		{
			name: "echo error with internal error in non-debug mode",
			err: &echo.HTTPError{
				Code:     http.StatusBadRequest,
				Message:  "bad request",
				Internal: errors.New("internal error details"),
			},
			debugMode: false,
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusBadRequest,
				Message:        "bad request",
				Code:           string(bacerrors.InternalError),
				Component:      "APIServer",
			},
		},
		{
			name:      "404 not found error with custom handling",
			err:       echo.NewHTTPError(http.StatusNotFound, "not found"),
			debugMode: false,
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusNotFound,
				Message:        "The requested resource/URL was not found.",
				Code:           string(bacerrors.NotFoundError),
				Component:      "APIServer",
				Hint:           "The server version may be older than this CLI version and does not support the endpoint yet.",
				Details: map[string]string{
					"cli_usage": "If you are using the CLI, try updating your server to a newer version or check if the command is supported in your server version.",
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e := echo.New()
			e.Debug = tt.debugMode
			c := e.NewContext(req, rec)

			CustomHTTPErrorHandler(tt.err, c)

			suite.Equal(tt.want.HTTPStatusCode, rec.Result().StatusCode)

			var apiError apimodels.APIError
			suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

			suite.Equal(tt.want.Message, apiError.Message)
			suite.Equal(tt.want.Code, apiError.Code)
			suite.Equal(tt.want.Component, apiError.Component)
			suite.Equal(tt.want.Hint, apiError.Hint)
			if tt.want.Details != nil {
				suite.Equal(tt.want.Details, apiError.Details)
			}
		})
	}
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestDefaultError() {
	tests := []struct {
		name      string
		err       error
		debugMode bool
		want      apimodels.APIError
	}{
		{
			name:      "unknown error in non-debug mode",
			err:       errors.New("unknown error"),
			debugMode: false,
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusInternalServerError,
				Message:        "Internal server error",
				Code:           string(bacerrors.InternalError),
				Component:      "Unknown",
			},
		},
		{
			name:      "unknown error in debug mode",
			err:       errors.New("detailed error info"),
			debugMode: true,
			want: apimodels.APIError{
				HTTPStatusCode: http.StatusInternalServerError,
				Message:        "Internal server error. detailed error info",
				Code:           string(bacerrors.InternalError),
				Component:      "Unknown",
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			e := echo.New()
			e.Debug = tt.debugMode
			c := e.NewContext(req, rec)

			CustomHTTPErrorHandler(tt.err, c)

			suite.Equal(tt.want.HTTPStatusCode, rec.Result().StatusCode)

			var apiError apimodels.APIError
			suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

			suite.Equal(tt.want.Message, apiError.Message)
			suite.Equal(tt.want.Code, apiError.Code)
			suite.Equal(tt.want.Component, apiError.Component)
		})
	}
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestCommittedResponse() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	// Simulate a committed response
	c.Response().WriteHeader(http.StatusOK)
	suite.True(c.Response().Committed)

	err := errors.New("test error")
	CustomHTTPErrorHandler(err, c)

	// Should maintain the original status code
	suite.Equal(http.StatusOK, rec.Result().StatusCode)
	suite.Empty(rec.Body.String())
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestHeadRequest() {
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	err := errors.New("test error")

	CustomHTTPErrorHandler(err, c)

	suite.Equal(http.StatusInternalServerError, rec.Result().StatusCode)
	suite.Empty(rec.Body.String())
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestRequestIDPropagation() {
	tests := []struct {
		name      string
		requestID string
		err       error
	}{
		{
			name:      "with request ID",
			requestID: "test-request-id",
			err:       errors.New("test error"),
		},
		{
			name:      "without request ID",
			requestID: "",
			err:       errors.New("test error"),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.requestID != "" {
				req.Header.Set(echo.HeaderXRequestID, tt.requestID)
			}
			rec := httptest.NewRecorder()
			c := suite.echo.NewContext(req, rec)

			CustomHTTPErrorHandler(tt.err, c)

			var apiError apimodels.APIError
			suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

			suite.Equal(tt.requestID, apiError.RequestID)
		})
	}
}

func TestCustomHTTPErrorHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CustomHTTPErrorHandlerTestSuite))
}
