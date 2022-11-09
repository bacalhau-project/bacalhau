package computenode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
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

// Before each test
func (suite *ComputeNodeJobSelectionSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

// TestJobSelectionNoVolumes tests that when we have RejectStatelessJobs
// turned on we don't accept a job with no volumes
// but when it's not turned on the job is actually selected
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionNoVolumes() {
	ctx := context.Background()
	runTest := func(rejectSetting, expectedResult bool) {
		stack := testutils.NewNoopStack(ctx, suite.T(), computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				RejectStatelessJobs: rejectSetting,
			},
		}, noop_executor.ExecutorConfig{})
		defer stack.Node.CleanupManager.Cleanup()

		result, _, err := stack.Node.ComputeNode.SelectJob(ctx, GetProbeData(""))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	suite.Run("reject", func() { runTest(true, false) })
	suite.Run("accept", func() { runTest(false, true) })
}

// JobSelectionLocality tests that data locality is respected
// when selecting a job
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionLocality() {
	ctx := context.Background()
	runTest := func(locality computenode.JobSelectionDataLocality, shouldAddData, expectedResult bool) {
		stack := testutils.NewNoopStack(ctx, suite.T(),
			computenode.ComputeNodeConfig{
				JobSelectionPolicy: computenode.JobSelectionPolicy{
					Locality: locality,
				},
			},
			noop_executor.ExecutorConfig{
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					HasStorageLocally: func(context.Context, model.StorageSpec) (bool, error) { return shouldAddData, nil },
				},
			},
		)

		computeNode, cm := stack.Node.ComputeNode, stack.Node.CleanupManager
		defer cm.Cleanup()

		result, _, err := computeNode.SelectJob(ctx, GetProbeData("abc"))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	// we are local - we do have the file - we should accept
	suite.Run("local with file", func() { runTest(computenode.Local, true, true) })

	// we are local - we don't have the file - we should reject
	suite.Run("local without file", func() { runTest(computenode.Local, false, false) })

	// // we are anywhere - we do have the file - we should accept
	suite.Run("anywhere with file", func() { runTest(computenode.Anywhere, true, true) })

	// // we are anywhere - we don't have the file - we should accept
	suite.Run("anywhere without file", func() { runTest(computenode.Anywhere, false, true) })
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

		stack := testutils.NewNoopStack(ctx, suite.T(), computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				ProbeHTTP: svr.URL,
			},
		}, noop_executor.ExecutorConfig{})
		computeNode, cm := stack.Node.ComputeNode, stack.Node.CleanupManager
		defer cm.Cleanup()

		result, _, err := computeNode.SelectJob(ctx, GetProbeData(""))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	suite.Run("hook says no - we don't accept", func() {
		runTest(true, false)
	})
	suite.Run("hook says yes - we accept", func() {
		runTest(false, true)
	})
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
		stack := testutils.NewNoopStack(ctx, suite.T(), computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				ProbeExec: command,
			},
		}, noop_executor.ExecutorConfig{})
		computeNode, cm := stack.Node.ComputeNode, stack.Node.CleanupManager
		defer cm.Cleanup()

		result, _, err := computeNode.SelectJob(ctx, GetProbeData(""))
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), result, expectedResult)
	}

	suite.Run("hook says no - we don't accept", func() {
		runTest(true, false)
	})
	suite.Run("hook says yes - we accept", func() {
		runTest(false, true)
	})
}
