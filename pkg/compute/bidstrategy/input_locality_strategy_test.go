//go:build unit || !integration

package bidstrategy

import (
	"context"
	"testing"

	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/suite"
)

type InputLocalityStrategySuite struct {
	suite.Suite
	statelessJob BidStrategyRequest
	statefulJob  BidStrategyRequest
}

func (s *InputLocalityStrategySuite) SetupSuite() {
	s.statelessJob = getBidStrategyRequest()
	s.statefulJob = getBidStrategyRequestWithInput()
}

func (s *InputLocalityStrategySuite) TestInputLocality() {
	testCases := []struct {
		name              string
		policy            model.JobSelectionDataLocality
		hasStorageLocally bool
		expectedShouldBid bool
		request           BidStrategyRequest
	}{
		// we are local - we do have the file - we should accept
		{
			"local mode -> have file -> should accept",
			model.Local,
			true,
			true,
			s.statefulJob,
		},

		// we are local - we don't have the file - we should reject
		{
			"local mode -> don't have file -> should reject",
			model.Local,
			false,
			false,
			s.statefulJob,
		},

		// we are local - stateless job - we should accept
		{
			"local mode -> stateless job -> should accept",
			model.Local,
			false,
			true,
			s.statelessJob,
		},

		// we are anywhere - we do have the file - we should accept
		{
			"anywhere mode -> have file -> should accept",
			model.Anywhere,
			true,
			true,
			s.statefulJob,
		},

		// we are anywhere - we don't have the file - we should accept
		{
			"anywhere mode -> don't have file -> should accept",
			model.Anywhere,
			false,
			true,
			s.statefulJob,
		},

		// we are anywhere - stateless job - we should accept
		{
			"anywhere mode ->s tateless job -> should accept",
			model.Anywhere,
			false,
			true,
			s.statelessJob,
		},
	}

	for _, test := range testCases {
		s.Run(test.name, func() {
			executor := noop_executor.NewNoopExecutorWithConfig(noop_executor.ExecutorConfig{
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					HasStorageLocally: func(ctx context.Context, volume model.StorageSpec) (bool, error) {
						return test.hasStorageLocally, nil
					},
				},
			})
			params := InputLocalityStrategyParams{
				Locality:  test.policy,
				Executors: noop_executor.NewNoopExecutorProvider(executor),
			}
			strategy := NewInputLocalityStrategy(params)
			result, err := strategy.ShouldBid(context.Background(), test.request)
			s.NoError(err)
			s.Equal(test.expectedShouldBid, result.ShouldBid)
		})
	}
}

func TestInputLocalityStrategySuite(t *testing.T) {
	suite.Run(t, new(InputLocalityStrategySuite))
}
