//go:build integration || !unit

package devstack

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"

	"github.com/filecoin-project/bacalhau/pkg/logger"
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

// Before each test
func (suite *DevstackSubmitSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	err := system.InitConfigForTesting(suite.T())
	require.NoError(suite.T(), err)
}

func (suite *DevstackSubmitSuite) TestEmptySpec() {
	ctx := context.Background()

	stack, cm := testutils.SetupTest(
		ctx,
		suite.T(),

		1,
		0,
		false,
		node.NewComputeConfigWithDefaults(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, t, "pkg/test/devstack/submittest/testemptyspec")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	j := &model.Job{}
	j.Deal = model.Deal{Concurrency: 1}
	_, missingSpecError := apiClient.Submit(ctx, j, nil)

	require.Error(suite.T(), missingSpecError)

	j = &model.Job{}
	j.Spec = model.Spec{Engine: model.EngineDocker}
	_, missingDealError := apiClient.Submit(ctx, j, nil)
	require.Error(suite.T(), missingDealError)
}
