package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/assert"
)

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func TestSelectAllJobs(t *testing.T) {
	t.Skip("TEMP_SKIP_FOR_NULL_POINTER_FAST_FAIL_TEST")

	type TestCase struct {
		name            string
		policy          computenode.JobSelectionPolicy
		nodeCount       int
		addFilesCount   int
		expectedAccepts int
	}

	runTest := func(testCase TestCase) {
		ctx, span := newSpan(testCase.name)
		defer span.End()
		scenario := scenario.CatFileToStdout(t)
		stack, cm := SetupTest(t, testCase.nodeCount, 0, computenode.ComputeNodeConfig{
			JobSelectionPolicy: testCase.policy,
		})
		defer TeardownTest(stack, cm)

		nodeIds, err := stack.GetNodeIds()
		assert.NoError(t, err)

		inputStorageList, err := scenario.SetupStorage(stack, storage.IPFSAPICopy, testCase.addFilesCount)
		assert.NoError(t, err)

		jobSpec := &executor.JobSpec{
			Engine:   executor.EngineDocker,
			Verifier: verifier.VerifierIpfs,
			Docker:   scenario.GetJobSpec(),
			Inputs:   inputStorageList,
			Outputs:  scenario.Outputs,
		}

		jobDeal := &executor.JobDeal{
			Concurrency: testCase.nodeCount,
		}

		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)
		submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
		assert.NoError(t, err)

		// wait for the job to complete across all nodes
		err = stack.WaitForJob(ctx, submittedJob.ID,
			devstack.WaitForJobThrowErrors([]executor.JobStateType{
				executor.JobStateBidRejected,
				executor.JobStateError,
			}),
			devstack.WaitForJobAllHaveState(nodeIds[0:testCase.expectedAccepts], executor.JobStateComplete),
		)

		assert.NoError(t, err)
	}

	for _, testCase := range []TestCase{

		// the default policy with all files added should end up with all jobs accepted
		{
			name:            "all nodes added files, all nodes ran job",
			policy:          computenode.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   3,
			expectedAccepts: 3,
		},

		// // check we get only 2 when we've only added data to 2
		{
			name:            "only nodes we added data to ran the job",
			policy:          computenode.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   2,
			expectedAccepts: 2,
		},

		// // check we run on all 3 nodes even though we only added data to 1
		{
			name: "only added files to 1 node but all 3 run it",
			policy: computenode.JobSelectionPolicy{
				Locality: computenode.Anywhere,
			},
			nodeCount:       3,
			addFilesCount:   1,
			expectedAccepts: 3,
		},
	} {
		runTest(testCase)
	}
}
