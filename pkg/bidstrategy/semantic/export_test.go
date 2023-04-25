//go:build unit || !integration

package semantic_test

import (
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func getBidStrategyRequest() bidstrategy.BidStrategyRequest {
	return bidstrategy.BidStrategyRequest{
		NodeID: "node-id",
		Job: model.Job{
			Metadata: model.Metadata{
				ID: "job-id",
			},
			Spec: model.Spec{
				Engine: model.EngineNoop,
			},
		},
	}
}

func getBidStrategyRequestWithInput() bidstrategy.BidStrategyRequest {
	request := getBidStrategyRequest()
	request.Job.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           "volume-id",
		},
	}
	return request
}
