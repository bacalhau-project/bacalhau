//go:build unit || !integration

package bidstrategy

import "github.com/bacalhau-project/bacalhau/pkg/model"

func getBidStrategyRequest() BidStrategyRequest {
	return BidStrategyRequest{
		NodeID: "node-id",
		Job: model.Job{
			Metadata: model.Metadata{
				ID: "job-id",
			},
			Spec: model.Spec{
				EngineSpec: model.EngineSpec{
					Type: model.EngineNoop,
					Spec: make(map[string]interface{}),
				},
			},
		},
	}
}

func getBidStrategyRequestWithInput() BidStrategyRequest {
	request := getBidStrategyRequest()
	request.Job.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           "volume-id",
		},
	}
	return request
}
