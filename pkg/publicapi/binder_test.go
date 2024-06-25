//go:build unit || !integration

package publicapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
)

// Mock struct that implements normalizable interface
type mockNormalizableRequest struct {
	Data string `json:"data" validate:"required"`
}

func (mr *mockNormalizableRequest) Normalize() {
	mr.Data = strings.TrimSpace(mr.Data)
}

// Mock struct that does not implement normalizable interface
type mockNonNormalizableRequest struct {
	Data string `json:"data" validate:"required"`
}

// TestSuite struct for CustomBinder
type CustomBinderTestSuite struct {
	suite.Suite
	e      *echo.Echo
	binder *CustomBinder
	rec    *httptest.ResponseRecorder
}

// SetupTest sets up the test environment
func (s *CustomBinderTestSuite) SetupTest() {
	s.e = echo.New()
	s.binder = NewCustomBinder()
	s.rec = httptest.NewRecorder()
}

// TestBindWithNormalization tests binding with normalization
func (s *CustomBinderTestSuite) TestBindWithNormalization() {
	echoContext := s.mockRequest(`{"data": " some data "}`)
	mockReq := new(mockNormalizableRequest)
	err := s.binder.Bind(mockReq, echoContext)

	s.NoError(err)
	s.Equal("some data", mockReq.Data)
}

// TestBindWithoutNormalization tests binding without normalization
func (s *CustomBinderTestSuite) TestBindWithoutNormalization() {
	echoContext := s.mockRequest(`{"data": " some data "}`)

	mockReq := new(mockNonNormalizableRequest)
	err := s.binder.Bind(mockReq, echoContext)

	s.NoError(err)
	s.Equal(" some data ", mockReq.Data)
}

// TestBindWithBadJSON tests binding with bad JSON
func (s *CustomBinderTestSuite) TestBindWithBadJSON() {
	echoContext := s.mockRequest(`{"data": " some data "`)

	mockReq := new(mockNormalizableRequest)
	err := s.binder.Bind(mockReq, echoContext)

	s.Error(err)
	s.Equal(400, err.(*echo.HTTPError).Code)
	s.Equal("unexpected EOF", err.(*echo.HTTPError).Message)
}

func (s *CustomBinderTestSuite) mockRequest(body string) echo.Context {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return s.e.NewContext(req, s.rec)
}

// TestCustomBinderSuite runs the test suite
func TestCustomBinderSuite(t *testing.T) {
	suite.Run(t, new(CustomBinderTestSuite))
}
