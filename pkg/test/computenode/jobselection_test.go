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
	"github.com/stretchr/testify/require"
)

// test that when we have RejectStatelessJobs turned on
// we don't accept a job with no volumes
// but when it's not turned on the job is actually selected
func TestJobSelectionNoVolumes(t *testing.T) {
	runTest := func(rejectSetting, expectedResult bool) {
		computeNode, _, cm := SetupTestNoop(t, computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				RejectStatelessJobs: rejectSetting,
			},
		}, noop_executor.ExecutorConfig{})
		defer cm.Cleanup()

		result, err := computeNode.SelectJob(context.Background(), GetProbeData(""))
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	runTest(true, false)
	runTest(false, true)
}

func TestJobSelectionLocality(t *testing.T) {

	// get the CID so we can use it in the tests below but without it actually being
	// added to the server (so we can test locality anywhere)
	EXAMPLE_TEXT := "hello"
	config.SetVolumeSizeRequestTimeout(2)
	cid, err := (func() (string, error) {
		_, ipfsStack, cm := SetupTestDockerIpfs(t, computenode.NewDefaultComputeNodeConfig())
		defer cm.Cleanup()
		return ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	}())
	require.NoError(t, err)

	runTest := func(locality computenode.JobSelectionDataLocality, shouldAddData, expectedResult bool) {

		computeNode, ipfsStack, cm := SetupTestDockerIpfs(t, computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality: locality,
			},
		})
		defer cm.Cleanup()

		if shouldAddData {
			_, err := ipfsStack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
			require.NoError(t, err)
		}

		result, err := computeNode.SelectJob(context.Background(), GetProbeData(cid))
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	// we are local - we do have the file - we should accept
	//runTest(computenode.Local, true, true)

	// we are local - we don't have the file - we should reject
	runTest(computenode.Local, false, false)

	// // we are anywhere - we do have the file - we should accept
	// runTest(computenode.Anywhere, true, true)

	// // we are anywhere - we don't have the file - we should accept
	// runTest(computenode.Anywhere, false, true)
}

func TestJobSelectionHttp(t *testing.T) {
	runTest := func(failMode, expectedResult bool) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, r.Method, "POST")
			if failMode {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 - Something bad happened!"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("200 - Everything is good!"))
			}

		}))
		defer svr.Close()

		computeNode, _, cm := SetupTestNoop(t, computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				ProbeHTTP: svr.URL,
			},
		}, noop_executor.ExecutorConfig{})
		defer cm.Cleanup()

		result, err := computeNode.SelectJob(context.Background(), GetProbeData(""))
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	runTest(true, false)
	runTest(false, true)
}

func TestJobSelectionExec(t *testing.T) {
	runTest := func(failMode, expectedResult bool) {
		command := "exit 0"
		if failMode {
			command = "exit 1"
		}
		computeNode, _, cm := SetupTestNoop(t, computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				ProbeExec: command,
			},
		}, noop_executor.ExecutorConfig{})
		defer cm.Cleanup()

		result, err := computeNode.SelectJob(context.Background(), GetProbeData(""))
		require.NoError(t, err)
		require.Equal(t, result, expectedResult)
	}

	runTest(true, false)
	runTest(false, true)
}

func TestJobSelectionEmptySpec(t *testing.T) {
	computeNode, _, cm := SetupTestNoop(t, computenode.ComputeNodeConfig{}, noop_executor.ExecutorConfig{})
	defer cm.Cleanup()

	_, err := computeNode.SelectJob(context.Background(), computenode.JobSelectionPolicyProbeData{
		NodeID: "test",
		JobID:  "test",
		Spec:   nil,
	})
	require.Error(t, err)
}
