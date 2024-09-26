//go:build unit || !integration

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
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	err := bacerrors.New("test base error").
		WithHTTPStatusCode(http.StatusBadRequest).
		WithCode("TEST_ERROR").
		WithComponent("TEST_COMPONENT")
	CustomHTTPErrorHandler(err, c)

	suite.Equal(http.StatusBadRequest, rec.Result().StatusCode)

	var apiError apimodels.APIError
	suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

	suite.Equal("test base error", apiError.Message)
	suite.Equal("TEST_ERROR", apiError.Code)
	suite.Equal("TEST_COMPONENT", apiError.Component)
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestEchoHTTPError() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	err := echo.NewHTTPError(http.StatusUnauthorized, "unauthorized access")

	CustomHTTPErrorHandler(err, c)

	suite.Equal(http.StatusUnauthorized, rec.Result().StatusCode)

	var apiError apimodels.APIError
	suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

	suite.Equal("unauthorized access", apiError.Message)
	suite.Equal(string(bacerrors.InternalError), apiError.Code)
	suite.Equal("APIServer", apiError.Component)
}

func (suite *CustomHTTPErrorHandlerTestSuite) TestDefaultError() {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	err := errors.New("unknown error")

	CustomHTTPErrorHandler(err, c)

	suite.Equal(http.StatusInternalServerError, rec.Result().StatusCode)

	var apiError apimodels.APIError
	suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

	suite.Equal("Internal server error", apiError.Message)
	suite.Equal(string(bacerrors.InternalError), apiError.Code)
	suite.Equal("Unknown", apiError.Component)
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
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderXRequestID, "test-request-id")
	rec := httptest.NewRecorder()
	c := suite.echo.NewContext(req, rec)

	err := errors.New("test error")

	CustomHTTPErrorHandler(err, c)

	suite.Equal(http.StatusInternalServerError, rec.Result().StatusCode)

	var apiError apimodels.APIError
	suite.Require().NoError(json.NewDecoder(rec.Body).Decode(&apiError))

	suite.Equal("test-request-id", apiError.RequestID)
}

func TestCustomHTTPErrorHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(CustomHTTPErrorHandlerTestSuite))
}
