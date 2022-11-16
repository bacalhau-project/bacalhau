//go:build unit || !integration

package bidstrategy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/require"
)

type StatelessJobStrategySuite struct {
	suite.Suite
	statelessJob BidStrategyRequest
	statefulJob  BidStrategyRequest
}

func (s *StatelessJobStrategySuite) SetupSuite() {
	s.statelessJob = getBidStrategyRequest()
	s.statefulJob = getBidStrategyRequestWithInput()
}

func (s *StatelessJobStrategySuite) TestRejectStateless_StatelessJob() {
	params := StatelessJobStrategyParams{RejectStatelessJobs: true}
	strategy := NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statelessJob)
	require.NoError(s.T(), err)
	require.False(s.T(), result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestRejectStateless_StatefulJob() {
	params := StatelessJobStrategyParams{RejectStatelessJobs: true}
	strategy := NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statefulJob)
	require.NoError(s.T(), err)
	require.True(s.T(), result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestAcceptStateless_StatelessJob() {
	params := StatelessJobStrategyParams{RejectStatelessJobs: false}
	strategy := NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statelessJob)
	require.NoError(s.T(), err)
	require.True(s.T(), result.ShouldBid)
}

func (s *StatelessJobStrategySuite) TestAcceptStateless_StatefulJob() {
	params := StatelessJobStrategyParams{RejectStatelessJobs: false}
	strategy := NewStatelessJobStrategy(params)

	result, err := strategy.ShouldBid(context.Background(), s.statefulJob)
	require.NoError(s.T(), err)
	require.True(s.T(), result.ShouldBid)
}

func TestStatelessJobStrategySuite(t *testing.T) {
	suite.Run(t, new(StatelessJobStrategySuite))
}
