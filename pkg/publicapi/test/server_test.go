//go:build unit || !integration

package test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/bacalhau-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServerSuite struct {
	suite.Suite
	server *publicapi.Server
	client *client.APIClient
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

// Before each test
func (s *ServerSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
}

// After each test
func (s *ServerSuite) TearDownTest() {
	if s.server != nil {
		err := s.server.Shutdown(context.Background())
		if err != nil {
			s.T().Log("Error shutting down server:", err)
		}
	}
}

func (s *ServerSuite) TestHealthz() {
	s.server, s.client = setupServer(s.T())
	rawHealthData := s.testEndpoint(s.T(), "/api/v1/healthz", "FreeSpace")

	var healthData types.HealthInfo
	err := marshaller.JSONUnmarshalWithMax(rawHealthData, &healthData)
	require.NoError(s.T(), err, "Error unmarshalling /healthz data.")

	// Checks that it's a number, and bigger than zero
	require.Greater(s.T(), int(healthData.DiskFreeSpace.ROOT.All), 0)

	// "all" should be bigger than "free" always
	require.Greater(s.T(), healthData.DiskFreeSpace.ROOT.All, healthData.DiskFreeSpace.ROOT.Free)
}

func (s *ServerSuite) TestLivez() {
	s.server, s.client = setupServer(s.T())
	_ = s.testEndpoint(s.T(), "/api/v1/livez", "OK")
}

func (s *ServerSuite) TestTimeout() {
	endpoint := "/timeout"
	timeout := 100 * time.Millisecond
	config := publicapi.NewConfig(publicapi.WithRequestHandlerTimeout(timeout))
	s.server, s.client = setupServerWithHandlers(s.T(), config, map[string]http.Handler{
		endpoint: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(timeout + 10*time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	})

	res, err := http.Get(s.client.BaseURI.JoinPath(endpoint).String())
	require.NoError(s.T(), err, "Could not get %s endpoint.", endpoint)
	require.Equal(s.T(), http.StatusServiceUnavailable, res.StatusCode)

	// validate response body
	body, err := io.ReadAll(res.Body)
	require.NoError(s.T(), err, "Could not read %s response body", endpoint)
	require.Equal(s.T(), body, []byte("Server Timeout!"))

	defer res.Body.Close()
}

func (s *ServerSuite) TestMaxBodyReader() {
	config := publicapi.NewConfig(publicapi.WithMaxBytesToReadInBody(500))
	s.server, s.client = setupServerWithConfig(s.T(), config)

	// Due to the rest of the Version payload we need MaxBytes minus
	// an amount that accounts for the other data we send
	payloadSize := int(500) - 16
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
			request := apimodels.VersionRequest{
				ClientID: strings.Repeat("a", tc.size),
			}

			var res apimodels.VersionResponse
			err := s.client.DoPost(context.Background(), "/api/v1/version", request, &res)
			if tc.expectError {
				require.Error(s.T(), err)
				if !strings.Contains(err.Error(), "Job not found") {
					if tc.expectError {
						require.Error(s.T(), err, "expected error")
						require.Contains(s.T(), err.Error(), "http: request body too large", "expected to error with body too large")
					} else {
						require.NoError(s.T(), err, "expected no error")
					}
				}
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ServerSuite) testEndpoint(t *testing.T, endpoint string, contentToCheck string) []byte {
	res, err := http.Get(s.client.BaseURI.JoinPath(endpoint).String())
	require.NoError(t, err, "Could not get %s endpoint.", endpoint)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err, "Could not read %s response body", endpoint)
	require.Contains(t, string(body), contentToCheck, "%s body does not contain '%s'.", endpoint, contentToCheck)
	return body
}
