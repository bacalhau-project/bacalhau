//go:build unit || !integration

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
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServerSuite struct {
	suite.Suite
	node   *node.Node
	client *publicapi.APIClient
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}

// Before each test
func (s *ServerSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	n, client := setupNodeForTest(s.T())
	s.node = n
	s.client = client
}

// After each test
func (s *ServerSuite) TearDownTest() {
	s.node.CleanupManager.Cleanup()
}

func (s *ServerSuite) TestList() {
	ctx := context.Background()

	// Should have no jobs initially:
	jobs, err := s.client.List(ctx, "", model.IncludeAny, model.ExcludeNone, 10, true, "created_at", true)
	require.NoError(s.T(), err)
	require.Empty(s.T(), jobs)

	// Submit a random job to the node:
	j := testutils.MakeNoopJob()

	_, err = s.client.Submit(ctx, j)
	require.NoError(s.T(), err)

	// Should now have one job:
	jobs, err = s.client.List(ctx, "", model.IncludeAny, model.ExcludeNone, 10, true, "created_at", true)
	require.NoError(s.T(), err)
	require.Len(s.T(), jobs, 1)
}

func (s *ServerSuite) TestSubmitRejectsJobWithSigilHeader() {
	j := testutils.MakeNoopJob()
	jobID, err := uuid.NewRandom()
	require.NoError(s.T(), err)

	s.client.DefaultHeaders["X-Bacalhau-Job-ID"] = jobID.String()
	_, err = s.client.Submit(context.Background(), j)
	require.Error(s.T(), err)
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
	config := publicapi.APIServerConfig{
		RequestHandlerTimeoutByURI: map[string]time.Duration{
			"/logz": 10 * time.Nanosecond,
		},
	}
	n, client := setupNodeForTestWithConfig(s.T(), config)
	s.node = n
	s.client = client

	endpoint := "/logz"
	res, err := http.Get(s.client.BaseURI + endpoint)
	require.NoError(s.T(), err, "Could not get %s endpoint.", endpoint)
	require.Equal(s.T(), http.StatusServiceUnavailable, res.StatusCode)

	// validate response body
	body, err := ioutil.ReadAll(res.Body)
	require.NoError(s.T(), err, "Could not read %s response body", endpoint)
	require.Equal(s.T(), body, []byte("Server Timeout!"))

	defer res.Body.Close()
}
func (s *ServerSuite) TestMaxBodyReader() {
	prev := publicapi.MaxBytesToReadInBody
	publicapi.MaxBytesToReadInBody = 500
	defer func() {
		publicapi.MaxBytesToReadInBody = prev
	}()

	// Due to the rest of the List payload we need MaxBytes minus
	// an amount that accounts for the other data we send
	maxSizeOfString := int(publicapi.MaxBytesToReadInBody) - 207
	testCases := []struct {
		name        string
		size        int
		expectError bool
	}{
		{name: "Max - 1", size: maxSizeOfString - 1, expectError: false},
		{name: "Max", size: maxSizeOfString, expectError: false},
		{name: "Max + 1", size: maxSizeOfString + 1, expectError: true},
	}

	_ = testCases

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			_, _, err := s.client.Get(context.TODO(), strings.Repeat("a", tc.size))
			if !strings.Contains(err.Error(), "Job not found") {
				if tc.expectError {
					require.Error(s.T(), err, "expected error")
					require.Contains(s.T(), err.Error(), "http: request body too large", "expected to error with body too large")
				} else {
					require.NoError(s.T(), err, "expected no error")
				}
			}
		})
	}
}

func (s *ServerSuite) testEndpoint(t *testing.T, endpoint string, contentToCheck string) []byte {

	res, err := http.Get(s.client.BaseURI + endpoint)
	require.NoError(t, err, "Could not get %s endpoint.", endpoint)
	defer res.Body.Close()

	require.Equal(t, res.StatusCode, http.StatusOK)
	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err, "Could not read %s response body", endpoint)
	require.Contains(t, string(body), contentToCheck, "%s body does not contain '%s'.", endpoint, contentToCheck)
	return body
}
