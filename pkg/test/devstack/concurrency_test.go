package devstack

import (
	"context"
	"testing"
	"time"

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

type DevstackConcurrencySuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackConcurrencySuite(t *testing.T) {
	suite.Run(t, new(DevstackConcurrencySuite))
}

// Before each test
func (suite *DevstackConcurrencySuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *DevstackConcurrencySuite) TestConcurrencyLimit() {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := context.Background()

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/concurrencytest")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	stack, cm := SetupTest(
		ctx,
		suite.T(),
		3,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)

	testCase := scenario.WasmHelloWorld()
	contexts, err := testCase.SetupContext(ctx, model.StorageSourceIPFS, devstack.ToIPFSClients(stack.Nodes[:3])...)
	require.NoError(suite.T(), err)

	// create a job
	j := &model.Job{}
	j.Spec = testCase.GetJobSpec()
	j.Spec.Verifier = model.VerifierNoop
	j.Spec.Publisher = model.PublisherNoop
	j.Spec.Contexts = contexts
	j.Spec.Outputs = testCase.Outputs
	j.Deal = model.Deal{
		Concurrency: 2,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	createdJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	stateChecker := func() error {
		return resolver.Wait(
			ctx,
			createdJob.ID,
			3,
			job.WaitThrowErrors([]model.JobStateType{
				model.JobStateError,
			}),
			job.WaitForJobStates(map[model.JobStateType]int{
				model.JobStateCompleted: 2,
			}),
		)
	}

	err = stateChecker()
	require.NoError(suite.T(), err)

	// wait a small time and then check again to make sure another JobStatePublished
	// did not sneak in afterwards
	time.Sleep(time.Second * 3)

	err = stateChecker()
	require.NoError(suite.T(), err)
}
