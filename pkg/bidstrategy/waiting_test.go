//go:build unit || !integration

package bidstrategy_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type waitingStrategyTestCase struct {
	waitOnBid, waitOnNoBid bool
	shouldBid, shouldWait  bool
	expectBid, expectWait  bool
}

func (t waitingStrategyTestCase) Name() string {
	return fmt.Sprintf(
		"should return %t bid %t wait when response is %t bid %t wait and waiting on bid %t waiting on no bid %t",
		t.expectBid,
		t.expectWait,
		t.shouldBid,
		t.shouldWait,
		t.waitOnBid,
		t.waitOnNoBid,
	)
}

func TestWaitsAppropriately(t *testing.T) {
	cases := []waitingStrategyTestCase{
		// Doesn't change existing answers when disabled
		{false, false, false, false, false, false},
		{false, false, false, true, false, true},
		{false, false, true, false, true, false},
		{false, false, true, true, true, true},
		// Sets waiting to true when positive response
		{true, false, true, false, true, true},
		{true, false, false, false, false, false},
		{true, false, true, true, true, true},
		{true, false, false, true, false, true},
		// Sets waiting to true when negative response
		{false, true, true, false, true, false},
		{false, true, false, false, false, true},
		{false, true, false, true, false, true},
		{false, true, true, true, true, true},
		// The above cases are independent
		{true, true, false, false, false, true},
		{true, true, true, false, true, true},
		{true, true, true, true, true, true},
		{true, true, false, true, false, true},
	}

	for _, testCase := range cases {
		underlying := bidstrategy.NewFixedBidStrategy(testCase.shouldBid, testCase.shouldWait)
		strategy := bidstrategy.NewWaitingStrategy(underlying, testCase.waitOnBid, testCase.waitOnNoBid)

		t.Run(testCase.Name()+"/ShouldBid", func(t *testing.T) {
			response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{})
			require.NoError(t, err)
			require.Equal(t, testCase.expectBid, response.ShouldBid)
			require.Equal(t, testCase.expectWait, response.ShouldWait)
		})

		t.Run(testCase.Name()+"/ShouldBidBasedOnUsage", func(t *testing.T) {
			response, err := strategy.ShouldBidBasedOnUsage(context.Background(), bidstrategy.BidStrategyRequest{}, model.ResourceUsageData{})
			require.NoError(t, err)
			require.Equal(t, testCase.expectBid, response.ShouldBid)
			require.Equal(t, testCase.expectWait, response.ShouldWait)
		})
	}
}
