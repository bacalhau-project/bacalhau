package devstack

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/logger"

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
func (suite *DevstackJobSelectionSuite) SetupSuite() {

}

// Before each test
func (suite *DevstackJobSelectionSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *DevstackJobSelectionSuite) TearDownTest() {

}

func (suite *DevstackJobSelectionSuite) TearDownSuite() {

}

// Re-use the docker executor tests but full end to end with libp2p transport
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
		ctx := context.Background()

		cm := system.NewCleanupManager()
		defer cm.Cleanup()

		t := system.GetTracer()
		ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/jobselectiontest/testselectalljobs")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		scenario := scenario.CatFileToStdout()
		stack, cm := SetupTest(ctx, suite.T(), testCase.nodeCount, 0, false, computenode.ComputeNodeConfig{
			JobSelectionPolicy: testCase.policy,
		}, requesternode.NewDefaultRequesterNodeConfig())

		nodeIDs, err := stack.GetNodeIds()
		require.NoError(suite.T(), err)

		inputStorageList, err := scenario.SetupStorage(ctx, model.StorageSourceIPFS, devstack.ToIPFSClients(stack.Nodes[:testCase.addFilesCount])...)
		require.NoError(suite.T(), err)

		j := &model.Job{}
		j.Spec = scenario.GetJobSpec()
		j.Spec.Inputs = inputStorageList
		j.Spec.Outputs = scenario.Outputs
		j.Deal = model.Deal{
			Concurrency: testCase.nodeCount,
		}

		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)
		submittedJob, err := apiClient.Submit(ctx, j, nil)
		require.NoError(suite.T(), err)

		resolver := apiClient.GetJobStateResolver()

		err = resolver.Wait(
			ctx,
			submittedJob.ID,
			len(nodeIDs),
			job.WaitDontExceedCount(testCase.expectedAccepts),
			job.WaitThrowErrors([]model.JobStateType{
				model.JobStateError,
			}),
			job.WaitForJobStates(map[model.JobStateType]int{
				model.JobStateCompleted: testCase.expectedAccepts,
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
		suite.Run(testCase.name, func() {
			runTest(testCase)
		})
	}
}
