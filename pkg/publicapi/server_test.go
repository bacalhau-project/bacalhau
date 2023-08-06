//go:build unit || !integration

package publicapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/types"
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
	s.Require().NoError(err, "Error unmarshalling /healthz data.")

	// Checks that it's a number, and bigger than zero
	s.Require().Greater(int(healthData.DiskFreeSpace.ROOT.All), 0)

	// "all" should be bigger than "free" always
	s.Require().Greater(healthData.DiskFreeSpace.ROOT.All, healthData.DiskFreeSpace.ROOT.Free)
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
	s.Require().NoError(err, "Error unmarshalling /varz data.")

}

func (s *ServerSuite) TestTimeout() {
	config := APIServerConfig{
		RequestHandlerTimeoutByURI: map[string]time.Duration{
			V1APIPrefix + "/logz": 10 * time.Nanosecond,
		},
	}
	s.client = setupNodeForTestWithConfig(s.T(), s.cleanupManager, config)

	endpoint := "/logz"
	res, err := http.Get(s.client.BaseURI.JoinPath(endpoint).String())
	s.Require().NoError(err, "Could not get %s endpoint.", endpoint)
	s.Require().Equal(http.StatusServiceUnavailable, res.StatusCode)

	// validate response body
	body, err := io.ReadAll(res.Body)
	s.Require().NoError(err, "Could not read %s response body", endpoint)
	s.Require().Equal(body, []byte("Server Timeout!"))

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
				s.Require().Error(err)
				if !strings.Contains(err.Error(), "Job not found") {
					if tc.expectError {
						s.Require().Error(err, "expected error")
						s.Require().Contains(err.Error(), "http: request body too large", "expected to error with body too large")
					} else {
						s.Require().NoError(err, "expected no error")
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
	s.Require().NoError(err, "Could not get %s endpoint.", endpoint)
	defer res.Body.Close()

	s.Require().Equal(res.StatusCode, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	s.Require().NoError(err, "Could not read %s response body", endpoint)
	s.Require().Contains(string(body), contentToCheck, "%s body does not contain '%s'.", endpoint, contentToCheck)
	return body
}
