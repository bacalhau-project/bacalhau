package publicapi

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
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

// Before all suite
func (suite *ServerSuite) SetupAllSuite() {
}

// Before each test
func (suite *ServerSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
}

func (suite *ServerSuite) TearDownTest() {
}

func (suite *ServerSuite) TearDownAllSuite() {
}

func (suite *ServerSuite) TestList() {
	ctx := context.Background()
	c, cm := SetupRequesterNodeForTests(suite.T())
	defer cm.Cleanup()

	// Should have no jobs initially:
	jobs, err := c.List(ctx, "", 10, true, "created_at", true)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), jobs)

	// Submit a random job to the node:
	j := MakeNoopJob()

	_, err = c.Submit(ctx, j, nil)
	require.NoError(suite.T(), err)

	// Should now have one job:
	jobs, err = c.List(ctx, "", 10, true, "created_at", true)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), jobs, 1)
}

func (suite *ServerSuite) TestHealthz() {
	rawHealthData := testEndpoint(suite.T(), "/healthz", "FreeSpace")

	var healthData types.HealthInfo
	err := json.Unmarshal(rawHealthData, &healthData)
	require.NoError(suite.T(), err, "Error unmarshalling /healthz data.")

	// Checks that it's a number, and bigger than zero
	require.Greater(suite.T(), int(healthData.DiskFreeSpace.ROOT.All), 0)

	// "all" should be bigger than "free" always
	require.Greater(suite.T(), healthData.DiskFreeSpace.ROOT.All, healthData.DiskFreeSpace.ROOT.Free)
}

func (suite *ServerSuite) TestLivez() {
	_ = testEndpoint(suite.T(), "/livez", "OK")
}

// TODO: #240 Should we test for /tmp/ipfs.log in tests?
// func (suite *ServerSuite) TestLogz() {
// 	_ = testEndpoint(suite.T(), "/logz", "OK")
// }

func (suite *ServerSuite) TestReadyz() {
	_ = testEndpoint(suite.T(), "/readyz", "READY")
}

func (suite *ServerSuite) TestVarz() {
	rawVarZBody := testEndpoint(suite.T(), "/varz", "{")

	var varZ types.VarZ
	err := json.Unmarshal(rawVarZBody, &varZ)
	require.NoError(suite.T(), err, "Error unmarshalling /varz data.")
}

func (suite *ServerSuite) TestTimeout() {
	config := &APIServerConfig{
		RequestHandlerTimeoutByURI: map[string]time.Duration{
			"/logz": 10 * time.Nanosecond,
		},
	}
	c, cm := SetupRequesterNodeForTestsWithConfig(suite.T(), config)
	defer cm.Cleanup()

	endpoint := "/logz"
	res, err := http.Get(c.BaseURI + endpoint)
	require.NoError(suite.T(), err, "Could not get %s endpoint.", endpoint)
	require.Equal(suite.T(), res.StatusCode, http.StatusServiceUnavailable)

	// validate response body
	body, err := ioutil.ReadAll(res.Body)
	require.NoError(suite.T(), err, "Could not read %s response body", endpoint)
	require.Equal(suite.T(), body, []byte("Server Timeout!"))

	defer res.Body.Close()
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
