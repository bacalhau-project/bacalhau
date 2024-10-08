//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/teststack"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
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
}

func (suite *DevstackSubmitSuite) TestEmptySpec() {
	ctx := context.Background()

	stack := testutils.Setup(ctx, suite.T(),
		devstack.WithNumberOfHybridNodes(1),
	)

	apiServer := stack.Nodes[0].APIServer
	apiClient := clientv2.New(fmt.Sprintf("http://%s:%d", apiServer.Address, apiServer.Port))

	j := &models.Job{}
	j.Count = 1
	resp, err := apiClient.Jobs().Put(ctx, &apimodels.PutJobRequest{
		Job: j,
	})

	suite.Require().Error(err)
	suite.Require().Empty(resp)

	j = &models.Job{}
	j.Tasks = []*models.Task{
		{
			Engine: &models.SpecConfig{
				Type:   models.EngineDocker,
				Params: make(map[string]interface{}),
			},
		},
	}
	resp, err = apiClient.Jobs().Put(ctx, &apimodels.PutJobRequest{
		Job: j,
	})
	suite.Require().Error(err)
	suite.Require().Empty(resp)
}
