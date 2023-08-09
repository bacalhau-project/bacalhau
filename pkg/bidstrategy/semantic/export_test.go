//go:build unit || !integration

package semantic_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func getBidStrategyRequest(t testing.TB) bidstrategy.BidStrategyRequest {
	spec, err := job.MakeSpec()
	if err != nil {
		t.Fatalf("failed to make spec: %s", err)
	}
	return bidstrategy.BidStrategyRequest{
		NodeID: "node-id",
		Job: model.Job{
			Metadata: model.Metadata{
				ID: "job-id",
			},
			Spec: spec,
		},
	}
}

func getBidStrategyRequestWithInput(t testing.TB) bidstrategy.BidStrategyRequest {
	request := getBidStrategyRequest(t)
	request.Job.Spec.Inputs = []model.StorageSpec{
		{
			StorageSource: model.StorageSourceIPFS,
			CID:           "volume-id",
		},
	}
	return request
}
