//go:build unit || !integration

package publicapi

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const testTimeout = 100 * time.Millisecond
const testMaxBytesToReadInBody = "500B"

type APIServerTestSuite struct {
	suite.Suite
	port   int
	server *Server
}

func (s *APIServerTestSuite) SetupTest() {
	port, err := network.GetFreePort()
	s.Require().NoError(err)
	s.port = port

	params := ServerParams{
		Router:  echo.New(),
		Address: "localhost",
		Port:    uint16(port),
		HostID:  "testHostID",
		Config: *NewConfig(
			WithRequestHandlerTimeout(testTimeout),
			WithMaxBytesToReadInBody(testMaxBytesToReadInBody),
		),
		Authorizer: authz.AlwaysAllow,
	}
	s.server, err = NewAPIServer(params)
	assert.NotNil(s.T(), s.server)
	assert.NoError(s.T(), err)
}

func (s *APIServerTestSuite) TearDownTest() {
	if s.server != nil {
		s.Require().NoError(s.server.Shutdown(context.Background()))
	}
}

func (s *APIServerTestSuite) TestGetURI() {
	uri := s.server.GetURI()
	assert.NotNil(s.T(), uri)
	assert.Equal(s.T(), "http", uri.Scheme)
	assert.Equal(s.T(), "localhost", uri.Hostname())
	assert.Equal(s.T(), strconv.Itoa(s.port), uri.Port())
}

func (s *APIServerTestSuite) TestListenAndServe() {
	s.Require().NoError(s.server.ListenAndServe(context.Background()))

	// Make a request to ensure the server is running
	resp, err := http.Get(s.server.GetURI().String())
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), resp)
	resp.Body.Close()

	// shutdown the server
	err = s.server.Shutdown(context.Background())
	assert.NoError(s.T(), err)

	// Make a request to ensure the server is shutdown
	_, err = http.Get(s.server.GetURI().String())
	assert.Error(s.T(), err)
}

func (s *APIServerTestSuite) TestTimeout() {
	// register slow handler
	endpoint := "/timeout"
	s.server.Router.GET(endpoint, func(c echo.Context) error {
		time.Sleep(testTimeout + 10*time.Millisecond)
		return c.String(http.StatusOK, "Ok!")
	})

	// start the server
	s.Require().NoError(s.server.ListenAndServe(context.Background()))

	res, err := http.Get(s.server.GetURI().JoinPath(endpoint).String())
	s.Require().NoError(err)
	defer func() { _ = res.Body.Close() }()
	s.validateResponse(res, http.StatusServiceUnavailable, "Server Timeout!")
}

func (s *APIServerTestSuite) TestMaxBodyReader() {
	// register handler
	endpoint := "/large"
	s.server.Router.POST(endpoint, func(c echo.Context) error {
		return c.String(http.StatusOK, "Ok!")
	})

	// start the server
	s.Require().NoError(s.server.ListenAndServe(context.Background()))

	payloadSize := 500
	testCases := []struct {
		name        string
		size        int
		expectError bool
	}{
		{name: "Max - 1", size: payloadSize - 1, expectError: false},
		{name: "Max", size: payloadSize, expectError: false},
		{name: "Max + 1", size: payloadSize + 1, expectError: true},
	}

	_ = testCases

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			request := strings.Repeat("a", tc.size)
			res, err := http.Post(s.server.GetURI().JoinPath(endpoint).String(), "text/plain", strings.NewReader(request))
			s.Require().NoError(err)
			defer func() { _ = res.Body.Close() }()
			if tc.expectError {
				s.validateResponse(res, http.StatusRequestEntityTooLarge, "Request Entity Too Large")
			} else {
				s.validateResponse(res, http.StatusOK, "Ok!")
			}
		})
	}
}

func (s *APIServerTestSuite) TestShutdownNotRunning() {
	// shutdown the server when it's not running should not error
	s.Require().NoError(s.server.Shutdown(context.Background()))
}

// validateResponse validates the response from the server
func (s *APIServerTestSuite) validateResponse(resp *http.Response, expectedStatusCode int, expectedBody string) {
	s.Require().NotNil(resp)
	s.Require().Equal(expectedStatusCode, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Require().Contains(string(body), expectedBody)
}

func TestAPIServerTestSuite(t *testing.T) {
	suite.Run(t, new(APIServerTestSuite))
}
