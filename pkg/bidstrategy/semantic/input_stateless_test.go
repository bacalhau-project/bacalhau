//go:build unit || !integration

package semantic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/require"

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
	require.NoError(s.T(), err)
	require.False(s.T(), result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestRejectStateless_StatefulJob() {
	params := semantic.StatelessJobStrategyParams{RejectStatelessJobs: true}
	strategy := semantic.NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statefulJob)
	require.NoError(s.T(), err)
	require.True(s.T(), result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestAcceptStateless_StatelessJob() {
	params := semantic.StatelessJobStrategyParams{RejectStatelessJobs: false}
	strategy := semantic.NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statelessJob)
	require.NoError(s.T(), err)
	require.True(s.T(), result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestAcceptStateless_StatefulJob() {
	params := semantic.StatelessJobStrategyParams{RejectStatelessJobs: false}
	strategy := semantic.NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statefulJob)
	require.NoError(s.T(), err)
	require.True(s.T(), result.ShouldBid)
}

func TestStatelessJobStrategySuite(t *testing.T) {
	suite.Run(t, new(StatelessJobStrategySuite))
}
