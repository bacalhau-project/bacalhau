package devstack

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	cmd "github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/rs/zerolog/log"
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
	ctx, span := newSpan("TestSimplestPythonWasmDashC")
	defer span.End()
	stack, cm := SetupTest(t, 1, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	nodeIds, err := stack.GetNodeIds()
	assert.NoError(t, err)

	// TODO: see also list_test.go, maybe factor out a common way to do this cli
	// setup
	_, out, err := cmd.ExecuteTestCobraCommand(t, cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"-c",
		"print(1+1)",
	)
	assert.NoError(t, err)

	jobId := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobId)
	// wait for the job to complete across all nodes
	err = stack.WaitForJob(ctx, jobId,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateBidRejected,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIds, executor.JobStateComplete),
	)
	assert.NoError(t, err)

	// load result from ipfs and check it
	// TODO: see devStackDockerStorageTest for how to do this

}

// TODO: test that > 10MB context is rejected

func TestSimplePythonWasm(t *testing.T) {
	ctx, span := newSpan("TestSimplePythonWasm")
	defer span.End()
	stack, cm := SetupTest(t, 1, 0, computenode.NewDefaultComputeNodeConfig())
	defer TeardownTest(stack, cm)

	nodeIds, err := stack.GetNodeIds()
	assert.NoError(t, err)
	tmpDir, err := ioutil.TempDir("", "devstack_test")
	assert.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tmpDir)
		assert.NoError(t, err)
	}()

	oldDir, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)
	defer func() {
		err := os.Chdir(oldDir)
		assert.NoError(t, err)
	}()

	// write bytes to main.py
	mainPy := []byte("print(1+1)")
	err = ioutil.WriteFile("main.py", mainPy, 0644)
	assert.NoError(t, err)

	_, out, err := cmd.ExecuteTestCobraCommand(t, cmd.RootCmd,
		fmt.Sprintf("--api-port=%d", stack.Nodes[0].APIServer.Port),
		"--api-host=localhost",
		"run",
		"python",
		"--deterministic",
		"main.py",
	)
	assert.NoError(t, err)
	jobId := strings.TrimSpace(out)
	log.Debug().Msgf("jobId=%s", jobId)
	time.Sleep(time.Second * 5)
	err = stack.WaitForJob(ctx, jobId,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateBidRejected,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIds, executor.JobStateComplete),
	)
	assert.NoError(t, err)

}

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
