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
	"github.com/stretchr/testify/require"
)

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func TestSelectAllJobs(t *testing.T) {

	t.Skip("https://github.com/filecoin-project/bacalhau/issues/361")

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
		require.NoError(t, err)

		inputStorageList, err := scenario.SetupStorage(stack, storage.IPFSAPICopy, testCase.addFilesCount)

		jobSpec := executor.JobSpec{
			Engine:   executor.EngineDocker,
			Verifier: verifier.VerifierIpfs,
			Docker:   scenario.GetJobSpec(),
			Inputs:   inputStorageList,
			Outputs:  scenario.Outputs,
		}

		jobDeal := executor.JobDeal{
			Concurrency: testCase.nodeCount,
		}

		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)
		submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
		require.NoError(t, err)

		// wait for the job to complete across all nodes
		err = stack.WaitForJobWithLogs(ctx, submittedJob.ID, true,
			devstack.WaitDontExceedCount(testCase.expectedAccepts),
			devstack.WaitForJobThrowErrors([]executor.JobStateType{
				executor.JobStateCancelled,
				executor.JobStateError,
			}),
			devstack.WaitForJobAllHaveState(nodeIds[0:testCase.expectedAccepts], executor.JobStateComplete),
		)

		require.NoError(t, err)
	}

	for _, testCase := range []TestCase{

		{
			name:            "all nodes added files, all nodes ran job",
			policy:          computenode.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   3,
			expectedAccepts: 3,
		},

		// check we get only 2 when we've only added data to 2
		{
			name:            "only nodes we added data to ran the job",
			policy:          computenode.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   2,
			expectedAccepts: 2,
		},

		// check we run on all 3 nodes even though we only added data to 1
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
