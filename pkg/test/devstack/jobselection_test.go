package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/assert"
)

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func TestSelectAllJobs(t *testing.T) {

	type TestCase struct {
		policy          compute_node.JobSelectionPolicy
		nodeCount       int
		addFilesCount   int
		expectedAccepts int
		expectedRejects int
	}

	runTest := func(testCase TestCase) {
		scenario := scenario.CatFileToStdout(t)
		stack, cm := SetupTest(t, testCase.nodeCount, 0, testCase.policy)
		defer TeardownTest(stack, cm)

		inputStorageList, err := scenario.SetupStorage(stack, storage.IPFS_API_COPY, testCase.addFilesCount)
		assert.NoError(t, err)

		jobSpec := &types.JobSpec{
			Engine:   string(executor.EXECUTOR_DOCKER),
			Verifier: string(verifier.VERIFIER_IPFS),
			Vm:       scenario.GetJobSpec(),
			Inputs:   inputStorageList,
			Outputs:  scenario.Outputs,
		}

		jobDeal := &types.JobDeal{
			Concurrency: testCase.nodeCount,
		}

		apiUri := stack.Nodes[0].ApiServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)
		submittedJob, err := apiClient.Submit(jobSpec, jobDeal)
		assert.NoError(t, err)

		// wait for the job to complete across all nodes
		err = stack.WaitForJob(submittedJob.Id, map[string]int{
			system.JOB_STATE_COMPLETE: testCase.expectedAccepts,
		}, []string{
			system.JOB_STATE_ERROR,
		})
		assert.NoError(t, err)
	}

	for _, testCase := range []TestCase{

		// the default policy with all files added should end up with all jobs accepted
		// {
		// 	policy:          compute_node.NewDefaultJobSelectionPolicy(),
		// 	nodeCount:       3,
		// 	addFilesCount:   3,
		// 	expectedAccepts: 3,
		// 	expectedRejects: 0,
		// },

		{
			policy:          compute_node.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   2,
			expectedAccepts: 2,
			expectedRejects: 0,
		},
	} {
		runTest(testCase)
	}
}
