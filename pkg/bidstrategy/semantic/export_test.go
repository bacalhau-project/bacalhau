//go:build unit || !integration

package semantic_test

import (
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func getBidStrategyRequest() bidstrategy.BidStrategyRequest {
	return bidstrategy.BidStrategyRequest{
		NodeID: "node-id",
		Job:    *mock.Job(),
	}
}

func getBidStrategyRequestWithInput() bidstrategy.BidStrategyRequest {
	request := getBidStrategyRequest()
	request.Job.Task().Artifacts = []*models.Artifact{
		{
			Source: models.NewSpecConfig(models.StorageSourceIPFS).WithParam("CID", "volume-id"),
			Target: "target",
		},
	}
	return request
}
