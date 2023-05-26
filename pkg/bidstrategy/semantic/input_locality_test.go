//go:build unit || !integration

package semantic_test

import (
	"context"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	testutil "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type InputLocalityStrategySuite struct {
	suite.Suite
	statelessJob bidstrategy.BidStrategyRequest
	statefulJob  bidstrategy.BidStrategyRequest
}

func (s *InputLocalityStrategySuite) SetupSuite() {
	inputs, err := (&ipfs.IPFSStorageSpec{CID: testutil.TestCID}).
		AsSpec("TODO", "TODO")
	s.Require().NoError(err)

	s.statelessJob = bidstrategy.BidStrategyRequest{}
	s.statefulJob = bidstrategy.BidStrategyRequest{
		Job: model.Job{
			Spec: model.Spec{
				Inputs: []spec.Storage{inputs},
			},
		},
	}
}

func (s *InputLocalityStrategySuite) TestInputLocality() {
	testCases := []struct {
		name              string
		policy            model.JobSelectionDataLocality
		hasStorageLocally bool
		expectedShouldBid bool
		request           bidstrategy.BidStrategyRequest
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
			noop_executor := noop_executor.NewNoopExecutorWithConfig(noop_executor.ExecutorConfig{
				ExternalHooks: noop_executor.ExecutorConfigExternalHooks{
					HasStorageLocally: func(ctx context.Context, volume spec.Storage) (bool, error) {
						return test.hasStorageLocally, nil
					},
				},
			})
			params := semantic.InputLocalityStrategyParams{
				Locality:  test.policy,
				Executors: model.NewNoopProvider[cid.Cid, executor.Executor](noop_executor),
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
