package devstack

import (
	"strings"
	"testing"

	cmd "github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// full end-to-end test of python/wasm:
//
// * use CLI to submit a python job with --deterministic set
// * context (python file and requirements.txt) should be pinned to ipfs by
//   requester node
// * docker executor downloads context and starts wasm container image with the
//   context mounted in

func TestSimplestPythonWasmDashC(t *testing.T) {
	_, out, err := cmd.ExecuteTestCobraCommand(t, cmd.RootCmd,
		"run",
		"python",
		"--deterministic",
		"-c",
		"print(1+1)",
	)
	assert.NoError(t, err)

	ctx, span := newSpan("TestSimplestPythonWasmDashC")
	defer span.End()
	stack, cm := SetupTest(t, 1, 0, compute_node.NewDefaultJobSelectionPolicy())
	defer TeardownTest(stack, cm)

	nodeIds, err := stack.GetNodeIds()
	assert.NoError(t, err)

	// XXX This might not work because run_python.go just uses fmt.Printf of the
	// job id. It needs to write it to the cobra buffer instead.

	jobId := strings.TrimSpace(out)
	// wait for the job to complete across all nodes
	err = stack.WaitForJob(ctx, jobId,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateBidRejected,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIds, executor.JobStateComplete),
	)
	assert.NoError(t, err)
	// load result from ipfs
	// result, err := stack.GetJobResult(ctx, jobId)
	// assert.NoError(t, err)
	// assert.Equal(t, "2", result)

}

// func TestSimplePythonWasm(t *testing.T) {
// 	tmpDir, err := ioutil.TempDir("", "devstack_test")
// 	assert.NoError(t, err)
// 	defer func() {
// 		err := os.RemoveAll(tmpDir)
// 		assert.NoError(t, err)
// 	}()

// 	oldDir, err := os.Getwd()
// 	assert.NoError(t, err)
// 	err = os.Chdir(tmpDir)
// 	assert.NoError(t, err)
// 	defer func() {
// 		err := os.Chdir(oldDir)
// 		assert.NoError(t, err)
// 	}()

// 	// write bytes to main.py
// 	mainPy := []byte("print(1+1)")
// 	err = ioutil.WriteFile("main.py", mainPy, 0644)
// 	assert.NoError(t, err)

// 	_, out, err := ExecuteTestCobraCommand(t, RootCmd,
// 		"run",
// 		"python",
// 		"--deterministic",
// 		"main.py",
// 	)
// 	assert.NoError(t, err)

// }

// func TestPythonWasmWithRequirements(t *testing.T) {
// 	tmpDir, err := ioutil.TempDir("", "devstack_test")
// 	assert.NoError(t, err)
// 	defer func() {
// 		err := os.RemoveAll(tmpDir)
// 		assert.NoError(t, err)
// 	}()

// 	oldDir, err := os.Getwd()
// 	assert.NoError(t, err)
// 	err = os.Chdir(tmpDir)
// 	assert.NoError(t, err)
// 	defer func() {
// 		err := os.Chdir(oldDir)
// 		assert.NoError(t, err)
// 	}()

// 	_, out, err := ExecuteTestCobraCommand(t, RootCmd,
// 		"run",
// 		"python",
// 		"--deterministic",
// 		"main.py",
// 	)
// 	assert.NoError(t, err)

// }

//////////////////////////////////////
// OLD

// func TestSelectAllJobs(t *testing.T) {

// 	type TestCase struct {
// 		name            string
// 		policy          compute_node.JobSelectionPolicy
// 		nodeCount       int
// 		addFilesCount   int
// 		expectedAccepts int
// 	}

// 	runTest := func(testCase TestCase) {
// 		ctx, span := newSpan(testCase.name)
// 		defer span.End()
// 		scenario := scenario.CatFileToStdout(t)
// 		stack, cm := SetupTest(t, testCase.nodeCount, 0, testCase.policy)
// 		defer TeardownTest(stack, cm)

// 		nodeIds, err := stack.GetNodeIds()
// 		assert.NoError(t, err)

// 		inputStorageList, err := scenario.SetupStorage(stack, storage.IPFS_API_COPY, testCase.addFilesCount)
// 		assert.NoError(t, err)

// 		jobSpec := &executor.JobSpec{
// 			Engine:   executor.EngineDocker,
// 			Verifier: verifier.VerifierIpfs,
// 			Vm:       scenario.GetJobSpec(),
// 			Inputs:   inputStorageList,
// 			Outputs:  scenario.Outputs,
// 		}

// 		jobDeal := &executor.JobDeal{
// 			Concurrency: testCase.nodeCount,
// 		}

// 		apiUri := stack.Nodes[0].ApiServer.GetURI()
// 		apiClient := publicapi.NewAPIClient(apiUri)
// 		submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal)
// 		assert.NoError(t, err)

// 		// wait for the job to complete across all nodes
// 		err = stack.WaitForJob(ctx, submittedJob.Id,
// 			devstack.WaitForJobThrowErrors([]executor.JobStateType{
// 				executor.JobStateBidRejected,
// 				executor.JobStateError,
// 			}),
// 			devstack.WaitForJobAllHaveState(nodeIds[0:testCase.expectedAccepts], executor.JobStateComplete),
// 		)

// 		assert.NoError(t, err)
// 	}

// 	for _, testCase := range []TestCase{

// 		// the default policy with all files added should end up with all jobs accepted
// 		{
// 			name:            "all nodes added files, all nodes ran job",
// 			policy:          compute_node.NewDefaultJobSelectionPolicy(),
// 			nodeCount:       3,
// 			addFilesCount:   3,
// 			expectedAccepts: 3,
// 		},

// 		// // check we get only 2 when we've only added data to 2
// 		{
// 			name:            "only nodes we added data to ran the job",
// 			policy:          compute_node.NewDefaultJobSelectionPolicy(),
// 			nodeCount:       3,
// 			addFilesCount:   2,
// 			expectedAccepts: 2,
// 		},

// 		// // check we run on all 3 nodes even though we only added data to 1
// 		{
// 			name: "only added files to 1 node but all 3 run it",
// 			policy: compute_node.JobSelectionPolicy{
// 				Locality: compute_node.Anywhere,
// 			},
// 			nodeCount:       3,
// 			addFilesCount:   1,
// 			expectedAccepts: 3,
// 		},
// 	} {
// 		runTest(testCase)
// 	}
// }
