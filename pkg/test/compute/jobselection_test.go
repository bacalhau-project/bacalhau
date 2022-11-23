//go:build unit || !integration

package compute

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
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
	err := system.InitConfigForTesting(suite.T())
	require.NoError(suite.T(), err)
}

// TestJobSelectionNoVolumes tests that when we have RejectStatelessJobs
// turned on we don't accept a job with no volumes
// but when it's not turned on the job is actually selected
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionNoVolumes() {
	ctx := context.Background()
	runTest := func(rejectSetting, expectedResult bool) {
		stack := testutils.NewNoopStack(ctx, suite.T(), node.NewComputeConfigWith(node.ComputeConfigParams{
			JobSelectionPolicy: model.JobSelectionPolicy{
				RejectStatelessJobs: rejectSetting,
			},
		}), noop_executor.ExecutorConfig{})
		defer stack.Node.CleanupManager.Cleanup()

		request := frontend.AskForBidRequest{
			Job:          *GetJob(""),
			ShardIndexes: []int{0},
		}

		result, err := stack.Node.ComputeNode.AskForBid(ctx, request)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), len(result.ShardResponse), 1)
		require.Equal(suite.T(), result.ShardResponse[0].Accepted, expectedResult)
	}

	suite.Run("reject", func() { runTest(true, false) })
	suite.Run("accept", func() { runTest(false, true) })
}

// JobSelectionLocality tests that data locality is respected
// when selecting a job
func (suite *ComputeNodeJobSelectionSuite) TestJobSelectionLocality() {
	ctx := context.Background()
	runTest := func(locality model.JobSelectionDataLocality, shouldAddData, expectedResult bool) {
		stack := testutils.NewNoopStack(ctx, suite.T(), node.NewComputeConfigWith(node.ComputeConfigParams{
			JobSelectionPolicy: model.JobSelectionPolicy{
				Locality: locality,
			},
		}),
			noop_executor.ExecutorConfig{
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					HasStorageLocally: func(context.Context, model.StorageSpec) (bool, error) { return shouldAddData, nil },
				},
			},
		)
		defer stack.Node.CleanupManager.Cleanup()

		request := frontend.AskForBidRequest{
			Job:          *GetJob("abc"),
			ShardIndexes: []int{0},
		}

		result, err := stack.Node.ComputeNode.AskForBid(ctx, request)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), len(result.ShardResponse), 1)
		require.Equal(suite.T(), result.ShardResponse[0].Accepted, expectedResult)
	}

	// we are local - we do have the file - we should accept
	suite.Run("local with file", func() { runTest(model.Local, true, true) })

	// we are local - we don't have the file - we should reject
	suite.Run("local without file", func() { runTest(model.Local, false, false) })

	// // we are anywhere - we do have the file - we should accept
	suite.Run("anywhere with file", func() { runTest(model.Anywhere, true, true) })

	// // we are anywhere - we don't have the file - we should accept
	suite.Run("anywhere without file", func() { runTest(model.Anywhere, false, true) })
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

		stack := testutils.NewNoopStack(ctx, suite.T(), node.NewComputeConfigWith(node.ComputeConfigParams{
			JobSelectionPolicy: model.JobSelectionPolicy{
				ProbeHTTP: svr.URL,
			},
		}), noop_executor.ExecutorConfig{})
		defer stack.Node.CleanupManager.Cleanup()

		request := frontend.AskForBidRequest{
			Job:          *GetJob(""),
			ShardIndexes: []int{0},
		}

		result, err := stack.Node.ComputeNode.AskForBid(ctx, request)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), len(result.ShardResponse), 1)
		require.Equal(suite.T(), result.ShardResponse[0].Accepted, expectedResult)
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
		stack := testutils.NewNoopStack(ctx, suite.T(), node.NewComputeConfigWith(node.ComputeConfigParams{
			JobSelectionPolicy: model.JobSelectionPolicy{
				ProbeExec: command,
			},
		}), noop_executor.ExecutorConfig{})
		defer stack.Node.CleanupManager.Cleanup()

		request := frontend.AskForBidRequest{
			Job:          *GetJob(""),
			ShardIndexes: []int{0},
		}

		result, err := stack.Node.ComputeNode.AskForBid(ctx, request)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), 1, len(result.ShardResponse))
		require.Equal(suite.T(), expectedResult, result.ShardResponse[0].Accepted)
	}

	suite.Run("hook says no - we don't accept", func() {
		runTest(true, false)
	})
	suite.Run("hook says yes - we accept", func() {
		runTest(false, true)
	})
}
