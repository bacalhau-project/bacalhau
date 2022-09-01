package computenode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/config"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ComputeNodeJobSelectionSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestComputeNodeJobSelectionSuite(t *testing.T) {
	suite.Run(t, new(ComputeNodeJobSelectionSuite))
}

// Before all suite
func (suite *ComputeNodeJobSelectionSuite) SetupAllSuite() {

}

// Before each test
func (suite *ComputeNodeJobSelectionSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *ComputeNodeJobSelectionSuite) TearDownTest() {
}

func (suite *ComputeNodeJobSelectionSuite) TearDownAllSuite() {

}

// TestJobSelectionNoVolumes tests that when we have RejectStatelessJobs
// turned on we don't accept a job with no volumes
// but when it's not turned on the job is actually selected
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionNoVolumes() {
	ctx := context.Background()
	runTest := func(rejectSetting, expectedResult bool) {
		stack := testutils.NewNoopStack(suite.T(), computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				RejectStatelessJobs: rejectSetting,
			},
		}, noop_executor.ExecutorConfig{})
		defer stack.CleanupManager.Cleanup()

		result, _, err := stack.ComputeNode.SelectJob(context.Background(), GetProbeData(""))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	runTest(true, false)
	runTest(false, true)
}

// JobSelectionLocality tests that data locality is respected
// when selecting a job
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionLocality() {
	ctx := context.Background()

	// get the CID so we can use it in the tests below but without it actually being
	// added to the server (so we can test locality anywhere)
	EXAMPLE_TEXT := "hello from job selection locality"
	config.SetVolumeSizeRequestTimeout(2)
	cid, err := (func() (string, error) {
		stack := testutils.NewDockerIpfsStack(suite.T(), computenode.NewDefaultComputeNodeConfig())
		ipfsStack, cm := stack.IpfsStack, stack.CleanupManager

		defer cm.Cleanup()
		return ipfsStack.AddTextToNodes(ctx, 1, []byte(EXAMPLE_TEXT))
	}())
	require.NoError(suite.T(), err)

	runTest := func(locality computenode.JobSelectionDataLocality, shouldAddData, expectedResult bool) {

		stack := testutils.NewDockerIpfsStack(suite.T(), computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality: locality,
			},
		})
		computeNode, ipfsStack, cm := stack.ComputeNode, stack.IpfsStack, stack.CleanupManager
		defer cm.Cleanup()

		if shouldAddData {
			_, err := ipfsStack.AddTextToNodes(ctx, 1, []byte(EXAMPLE_TEXT))
			require.NoError(suite.T(), err)
		}

		result, _, err := computeNode.SelectJob(ctx, GetProbeData(cid))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	// we are local - we do have the file - we should accept
	runTest(computenode.Local, true, true)

	// we are local - we don't have the file - we should reject
	runTest(computenode.Local, false, false)

	// // we are anywhere - we do have the file - we should accept
	runTest(computenode.Anywhere, true, true)

	// // we are anywhere - we don't have the file - we should accept
	runTest(computenode.Anywhere, false, true)
}

// TestJobSelectionHttp tests that we can select a job based on
// an http hook
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionHttp() {
	ctx := context.Background()
	runTest := func(failMode, expectedResult bool) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(suite.T(), r.Method, "POST")
			if failMode {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 - Something bad happened!"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("200 - Everything is good!"))
			}

		}))
		defer svr.Close()

		computeNode, _, _, cm := SetupTestNoop(suite.T(), computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				ProbeHTTP: svr.URL,
			},
		}, noop_executor.ExecutorConfig{})
		computeNode, cm := stack.ComputeNode, stack.CleanupManager
		defer cm.Cleanup()

		result, _, err := computeNode.SelectJob(ctx, GetProbeData(""))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	// hook says no - we don't accept
	runTest(true, false)
	// hook says yes - we accept
	runTest(false, true)
}

// TestJobSelectionExec tests that we can select a job based on
// an external command hook
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionExec() {
	ctx := context.Background()
	runTest := func(failMode, expectedResult bool) {
		command := "exit 0"
		if failMode {
			command = "exit 1"
		}
		computeNode, _, _, cm := SetupTestNoop(suite.T(), computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				ProbeExec: command,
			},
		}, noop_executor.ExecutorConfig{})
		computeNode, cm := stack.ComputeNode, stack.CleanupManager
		defer cm.Cleanup()

		result, _, err := computeNode.SelectJob(ctx, GetProbeData(""))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	// hook says no - we don't accept
	runTest(true, false)
	// hook says yes - we accept
	runTest(false, true)
}

// TestJobSelectionEmptySpec tests that a job with an empty spec is rejected
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionEmptySpec() {
	computeNode, _, _, cm := SetupTestNoop(suite.T(), computenode.ComputeNodeConfig{}, noop_executor.ExecutorConfig{})
	defer cm.Cleanup()

	_, _, err := computeNode.SelectJob(ctx, computenode.JobSelectionPolicyProbeData{
		NodeID: "test",
		JobID:  "test",
	})
	require.Error(suite.T(), err)
}
