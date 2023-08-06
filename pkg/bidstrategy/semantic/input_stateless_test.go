//go:build unit || !integration

package semantic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
)

type StatelessJobStrategySuite struct {
	suite.Suite
	statelessJob bidstrategy.BidStrategyRequest
	statefulJob  bidstrategy.BidStrategyRequest
}

func (s *StatelessJobStrategySuite) SetupSuite() {
	s.statelessJob = getBidStrategyRequest()
	s.statefulJob = getBidStrategyRequestWithInput()
}

func (s *StatelessJobStrategySuite) TestRejectStateless_StatelessJob() {
	params := semantic.StatelessJobStrategyParams{RejectStatelessJobs: true}
	strategy := semantic.NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statelessJob)
	s.Require().NoError(err)
	s.Require().False(result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestRejectStateless_StatefulJob() {
	params := semantic.StatelessJobStrategyParams{RejectStatelessJobs: true}
	strategy := semantic.NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statefulJob)
	s.Require().NoError(err)
	s.Require().True(result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestAcceptStateless_StatelessJob() {
	params := semantic.StatelessJobStrategyParams{RejectStatelessJobs: false}
	strategy := semantic.NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statelessJob)
	s.Require().NoError(err)
	s.Require().True(result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestAcceptStateless_StatefulJob() {
	params := semantic.StatelessJobStrategyParams{RejectStatelessJobs: false}
	strategy := semantic.NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statefulJob)
	s.Require().NoError(err)
	s.Require().True(result.ShouldBid)
}

func TestStatelessJobStrategySuite(t *testing.T) {
	suite.Run(t, new(StatelessJobStrategySuite))
}
