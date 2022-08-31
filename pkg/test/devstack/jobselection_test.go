package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DevstackJobSelectionSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackJobSelectionSuite(t *testing.T) {
	suite.Run(t, new(DevstackJobSelectionSuite))
}

// Before all suite
func (suite *DevstackJobSelectionSuite) SetupAllSuite() {

}

// Before each test
func (suite *DevstackJobSelectionSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *DevstackJobSelectionSuite) TearDownTest() {
}

func (suite *DevstackJobSelectionSuite) TearDownAllSuite() {

}

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func (suite *DevstackJobSelectionSuite) TestSelectAllJobs() {

	suite.T().Skip("https://github.com/filecoin-project/bacalhau/issues/361")

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
		scenario := scenario.CatFileToStdout(suite.T())
		stack, cm := SetupTest(suite.T(), testCase.nodeCount, 0, computenode.ComputeNodeConfig{
			JobSelectionPolicy: testCase.policy,
		})
		defer TeardownTest(stack, cm)

		nodeIDs, err := stack.GetNodeIds()
		require.NoError(suite.T(), err)

		inputStorageList, err := scenario.SetupStorage(stack, model.StorageSourceIPFS, testCase.addFilesCount)

		jobSpec := model.JobSpec{
			Engine:    model.EngineDocker,
			Verifier:  model.VerifierNoop,
			Publisher: model.PublisherNoop,
			Docker:    scenario.GetJobSpec(),
			Inputs:    inputStorageList,
			Outputs:   scenario.Outputs,
		}

		jobDeal := model.JobDeal{
			Concurrency: testCase.nodeCount,
		}

		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)
		submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
		require.NoError(suite.T(), err)

		resolver := apiClient.GetJobStateResolver()

		err = resolver.Wait(
			ctx,
			submittedJob.ID,
			len(nodeIDs),
			job.WaitDontExceedCount(testCase.expectedAccepts),
			job.WaitThrowErrors([]model.JobStateType{
				model.JobStateCancelled,
				model.JobStateError,
			}),
			job.WaitForJobStates(map[model.JobStateType]int{
				model.JobStatePublished: testCase.expectedAccepts,
			}),
		)
		require.NoError(suite.T(), err)
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
