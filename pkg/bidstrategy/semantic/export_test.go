//go:build unit || !integration

package semantic_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func getBidStrategyRequest(t testing.TB) bidstrategy.BidStrategyRequest {
	job := mock.Job()
	return bidstrategy.BidStrategyRequest{
		Job: *job,
	}
}

func getBidStrategyRequestWithInput(t testing.TB) bidstrategy.BidStrategyRequest {
	request := getBidStrategyRequest(t)
	request.Job.Task().InputSources = []*models.InputSource{
		{
			Source: models.NewSpecConfig(models.StorageSourceLocalDirectory).WithParam("SourcePath", "/dummy/path"),
			Target: "target",
		},
	}
	return request
}
