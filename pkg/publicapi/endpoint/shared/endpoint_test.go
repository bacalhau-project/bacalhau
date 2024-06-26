//go:build unit || !integration

package shared

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type EndpointSuite struct {
	suite.Suite
	router *echo.Echo
	e      *Endpoint
}

func (s *EndpointSuite) SetupSuite() {
	s.router = echo.New()
	s.e = NewEndpoint(EndpointParams{
		Router: s.router,
		NodeID: "testNodeID",
	})
}

func (s *EndpointSuite) TestEndpointId() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/id", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Equal("testNodeID", rr.Body.String())
}

func (s *EndpointSuite) TestEndpointVersion() {
	versionReq := &VersionRequest{ClientID: "testClient"}
	body, _ := json.Marshal(versionReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/version", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	expectedVersion := version.Get()
	expectedResponse, _ := json.Marshal(VersionResponse{VersionInfo: expectedVersion})

	s.Equal(http.StatusOK, rr.Code)
	s.Equal(string(expectedResponse), strings.TrimSpace(rr.Body.String()))
}

func (s *EndpointSuite) TestEndpointLivez() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/livez", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	s.Equal(http.StatusOK, rr.Code)
	s.Equal("OK", rr.Body.String())
}
func TestEndpointTestSuite(t *testing.T) {
	suite.Run(t, new(EndpointSuite))
}
