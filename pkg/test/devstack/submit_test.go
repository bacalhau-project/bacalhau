package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
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
	ctx, span := newSpan("TestEmptySpec")
	defer span.End()

	stack, cm := SetupTest(
		suite.T(),
		1,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	_, missingSpecError := apiClient.Submit(ctx, nil, &executor.JobDeal{}, nil)
	require.Error(suite.T(), missingSpecError)

	_, missingDealError := apiClient.Submit(ctx, &executor.JobSpec{}, nil, nil)
	require.Error(suite.T(), missingDealError)
}
