package devstack

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DevstackSubmitSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackSubmitSuite(t *testing.T) {
	suite.Run(t, new(DevstackSubmitSuite))
}

// Before all suite
func (suite *DevstackSubmitSuite) SetupAllSuite() {

}

// Before each test
func (suite *DevstackSubmitSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *DevstackSubmitSuite) TearDownTest() {
}

func (suite *DevstackSubmitSuite) TearDownAllSuite() {

}

func (suite *DevstackSubmitSuite) TestEmptySpec() {
	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		suite.T(),

		1,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/submittest/testemptyspec")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	_, missingSpecError := apiClient.Submit(ctx, model.JobSpec{}, model.JobDeal{
		Concurrency: 1,
	}, nil)

	require.Error(suite.T(), missingSpecError)

	_, missingDealError := apiClient.Submit(ctx, model.JobSpec{
		Engine: model.EngineDocker,
	}, model.JobDeal{}, nil)
	require.Error(suite.T(), missingDealError)
}
