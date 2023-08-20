//go:build unit || !integration

package semantic_test

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/noop"
)

type InputLocalityStrategySuite struct {
	suite.Suite
	statelessJob bidstrategy.BidStrategyRequest
	statefulJob  bidstrategy.BidStrategyRequest
}

func (s *InputLocalityStrategySuite) SetupSuite() {
	statefulJob := mock.Job()
	statefulJob.Task().InputSources = []*models.InputSource{
		{
			Source: models.NewSpecConfig(models.StorageSourceIPFS).WithParam("CID", "volume-id"),
			Target: "target",
		},
	}

	statelessJob := mock.Job()
	statelessJob.Task().InputSources = []*models.InputSource{}

	s.statelessJob = bidstrategy.BidStrategyRequest{
		Job: *statelessJob,
	}
	s.statefulJob = bidstrategy.BidStrategyRequest{
		Job: *statefulJob,
	}
}

func (s *InputLocalityStrategySuite) TestInputLocality() {
	testCases := []struct {
		name              string
		policy            semantic.JobSelectionDataLocality
		hasStorageLocally bool
		expectedShouldBid bool
		request           bidstrategy.BidStrategyRequest
	}{
		// we are local - we do have the file - we should accept
		{
			"local mode -> have file -> should accept",
			semantic.Local,
			true,
			true,
			s.statefulJob,
		},

		// we are local - we don't have the file - we should reject
		{
			"local mode -> don't have file -> should reject",
			semantic.Local,
			false,
			false,
			s.statefulJob,
		},

		// we are local - stateless job - we should accept
		{
			"local mode -> stateless job -> should accept",
			semantic.Local,
			false,
			true,
			s.statelessJob,
		},

		// we are anywhere - we do have the file - we should accept
		{
			"anywhere mode -> have file -> should accept",
			semantic.Anywhere,
			true,
			true,
			s.statefulJob,
		},

		// we are anywhere - we don't have the file - we should accept
		{
			"anywhere mode -> don't have file -> should accept",
			semantic.Anywhere,
			false,
			true,
			s.statefulJob,
		},

		// we are anywhere - stateless job - we should accept
		{
			"anywhere mode ->s tateless job -> should accept",
			semantic.Anywhere,
			false,
			true,
			s.statelessJob,
		},
	}

	for _, test := range testCases {
		s.Run(test.name, func() {
			fakeStorage := noop.NewNoopStorage()
			fakeStorage.Config.ExternalHooks.HasStorageLocally = func(ctx context.Context, volume models.InputSource) (bool, error) {
				return test.hasStorageLocally, nil
			}
			params := semantic.InputLocalityStrategyParams{
				Locality: test.policy,
				Storages: provider.NewNoopProvider[storage.Storage](fakeStorage),
			}
			strategy := semantic.NewInputLocalityStrategy(params)
			result, err := strategy.ShouldBid(context.Background(), test.request)
			s.NoError(err)
			s.Equal(test.expectedShouldBid, result.ShouldBid)
		})
	}
}

func TestInputLocalityStrategySuite(t *testing.T) {
	suite.Run(t, new(InputLocalityStrategySuite))
}
