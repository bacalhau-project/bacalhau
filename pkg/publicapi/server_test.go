//go:build !integration

package publicapi

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServerSuite struct {
	suite.Suite
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

func (s *ServerSuite) TestList() {
	ctx := context.Background()
	c, cm := SetupRequesterNodeForTests(s.T())
	defer cm.Cleanup()

	// Should have no jobs initially:
	jobs, err := c.List(ctx, "", 10, true, "created_at", true)
	require.NoError(s.T(), err)
	require.Empty(s.T(), jobs)

	// Submit a random job to the node:
	j := MakeNoopJob()

	_, err = c.Submit(ctx, j, nil)
	require.NoError(s.T(), err)

	// Should now have one job:
	jobs, err = c.List(ctx, "", 10, true, "created_at", true)
	require.NoError(s.T(), err)
	require.Len(s.T(), jobs, 1)
}

func (s *ServerSuite) TestHealthz() {
	rawHealthData := testEndpoint(s.T(), "/healthz", "FreeSpace")

	var healthData types.HealthInfo
	err := model.JSONUnmarshalWithMax(rawHealthData, &healthData)
	require.NoError(s.T(), err, "Error unmarshalling /healthz data.")

	// Checks that it's a number, and bigger than zero
	require.Greater(s.T(), int(healthData.DiskFreeSpace.ROOT.All), 0)

	// "all" should be bigger than "free" always
	require.Greater(s.T(), healthData.DiskFreeSpace.ROOT.All, healthData.DiskFreeSpace.ROOT.Free)
}

func (s *ServerSuite) TestLivez() {
	_ = testEndpoint(s.T(), "/livez", "OK")
}

// TODO: #240 Should we test for /tmp/ipfs.log in tests?
// func (s *ServerSuite) TestLogz() {
// 	_ = testEndpoint(s.T(), "/logz", "OK")
// }

func (s *ServerSuite) TestReadyz() {
	_ = testEndpoint(s.T(), "/readyz", "READY")
}

func (s *ServerSuite) TestVarz() {
	rawVarZBody := testEndpoint(s.T(), "/varz", "{")

	var varZ types.VarZ
	err := model.JSONUnmarshalWithMax(rawVarZBody, &varZ)
	require.NoError(s.T(), err, "Error unmarshalling /varz data.")

}

func (s *ServerSuite) TestTimeout() {
	config := &APIServerConfig{
		RequestHandlerTimeoutByURI: map[string]time.Duration{
			"/logz": 10 * time.Nanosecond,
		},
	}
	c, cm := SetupRequesterNodeForTestsWithConfig(s.T(), config)
	defer cm.Cleanup()

	endpoint := "/logz"
	res, err := http.Get(c.BaseURI + endpoint)
	require.NoError(s.T(), err, "Could not get %s endpoint.", endpoint)
	require.Equal(s.T(), res.StatusCode, http.StatusServiceUnavailable)

	// validate response body
	body, err := ioutil.ReadAll(res.Body)
	require.NoError(s.T(), err, "Could not read %s response body", endpoint)
	require.Equal(s.T(), body, []byte("Server Timeout!"))

	defer res.Body.Close()
}
func (s *ServerSuite) TestMaxBodyReader() {
	MaxBytesToReadInBody = 500

	c, cm := SetupRequesterNodeForTests(s.T())
	defer cm.Cleanup()

	// Due to headers we need MaxBytes minus 163
	maxSizeOfString := int(MaxBytesToReadInBody) - 163
	testCases := []struct {
		name        string
		size        int
		expectError bool
	}{
		{name: "Max - 1", size: maxSizeOfString - 1, expectError: false},
		{name: "Max", size: maxSizeOfString, expectError: false},
		{name: "Max + 1", size: maxSizeOfString + 1, expectError: true}}

	_ = testCases

	for _, tc := range testCases {
		_, _, err := c.Get(context.TODO(), strings.Repeat("a", tc.size))
		if !strings.Contains(err.Error(), "Job not found") {
			if tc.expectError {
				require.Error(s.T(), err, "%s: Expected error", tc.name)
				require.Contains(s.T(), err.Error(), "http: request body too large", "%s: Expected to error with body too large", tc.name)
			} else {
				require.NoError(s.T(), err, "%s: Expected no error", tc.name)
			}
		}
	}
}

func testEndpoint(t *testing.T, endpoint string, contentToCheck string) []byte {
	c, cm := SetupRequesterNodeForTests(t)
	defer cm.Cleanup()

	res, err := http.Get(c.BaseURI + endpoint)
	require.NoError(t, err, "Could not get %s endpoint.", endpoint)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, http.StatusOK)
	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err, "Could not read %s response body", endpoint)
	require.Contains(t, string(body), contentToCheck, "%s body does not contain '%s'.", endpoint, contentToCheck)
	return body
}
