//go:build unit || !integration

package publicapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServerSuite struct {
	suite.Suite
	cleanupManager *system.CleanupManager
	client         *APIClient
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

// Before each test
func (s *ServerSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	s.cleanupManager = system.NewCleanupManager()
	s.client = setupNodeForTest(s.T(), s.cleanupManager)
}

// After each test
func (s *ServerSuite) TearDownTest() {
	s.cleanupManager.Cleanup(context.Background())
}

func (s *ServerSuite) TestHealthz() {
	rawHealthData := s.testEndpoint(s.T(), "/healthz", "FreeSpace")

	var healthData types.HealthInfo
	err := model.JSONUnmarshalWithMax(rawHealthData, &healthData)
	require.NoError(s.T(), err, "Error unmarshalling /healthz data.")

	// Checks that it's a number, and bigger than zero
	require.Greater(s.T(), int(healthData.DiskFreeSpace.ROOT.All), 0)

	// "all" should be bigger than "free" always
	require.Greater(s.T(), healthData.DiskFreeSpace.ROOT.All, healthData.DiskFreeSpace.ROOT.Free)
}

func (s *ServerSuite) TestLivez() {
	_ = s.testEndpoint(s.T(), "/livez", "OK")
}

// TODO: #240 Should we test for /tmp/ipfs.log in tests?
// func (s *ServerSuite) TestLogz() {
// 	_ = s.testEndpoint(s.T(), "/logz", "OK")
// }

func (s *ServerSuite) TestReadyz() {
	_ = s.testEndpoint(s.T(), "/readyz", "READY")
}

func (s *ServerSuite) TestVarz() {
	rawVarZBody := s.testEndpoint(s.T(), "/varz", "{")

	var varZ types.VarZ
	err := model.JSONUnmarshalWithMax(rawVarZBody, &varZ)
	require.NoError(s.T(), err, "Error unmarshalling /varz data.")

}

func (s *ServerSuite) TestTimeout() {
	config := APIServerConfig{
		RequestHandlerTimeoutByURI: map[string]time.Duration{
			"/logz": 10 * time.Nanosecond,
		},
	}
	s.client = setupNodeForTestWithConfig(s.T(), s.cleanupManager, config)

	endpoint := "/logz"
	res, err := http.Get(s.client.BaseURI + endpoint)
	require.NoError(s.T(), err, "Could not get %s endpoint.", endpoint)
	require.Equal(s.T(), http.StatusServiceUnavailable, res.StatusCode)

	// validate response body
	body, err := io.ReadAll(res.Body)
	require.NoError(s.T(), err, "Could not read %s response body", endpoint)
	require.Equal(s.T(), body, []byte("Server Timeout!"))

	defer res.Body.Close()
}
func (s *ServerSuite) TestMaxBodyReader() {
	config := APIServerConfig{
		MaxBytesToReadInBody: 500,
	}
	s.client = setupNodeForTestWithConfig(s.T(), s.cleanupManager, config)

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
			request := VersionRequest{
				ClientID: strings.Repeat("a", tc.size),
			}

			var res VersionResponse
			err := s.client.Post(context.Background(), "version", request, &res)
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

	res, err := http.Get(s.client.BaseURI + endpoint)
	require.NoError(t, err, "Could not get %s endpoint.", endpoint)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err, "Could not read %s response body", endpoint)
	require.Contains(t, string(body), contentToCheck, "%s body does not contain '%s'.", endpoint, contentToCheck)
	return body
}
