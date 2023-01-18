package bidstrategy

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
)

type networkingStrategyTestCase struct {
	accept         bool
	job_networking model.NetworkConfig
	should_bid     bool
}

func (test networkingStrategyTestCase) String() string {
	return fmt.Sprintf(
		"should bid is %t when job requires %s and strategy accepts networking is %t",
		test.should_bid,
		test.job_networking.Type,
		test.accept,
	)
}

var networkingStrategyTestCases = []networkingStrategyTestCase{
	{false, model.NetworkConfig{Type: model.NetworkNone}, true},
	{false, model.NetworkConfig{Type: model.NetworkFull}, false},
	{true, model.NetworkConfig{Type: model.NetworkNone}, true},
	{true, model.NetworkConfig{Type: model.NetworkFull}, true},
}

func TestNetworkingStrategy(t *testing.T) {
	for _, test := range networkingStrategyTestCases {
		strategy := NewNetworkingStrategy(test.accept)
		request := BidStrategyRequest{
			Job: model.Job{
				Spec: model.Spec{Network: test.job_networking},
			},
		}

		t.Run("ShouldBid/"+test.String(), func(t *testing.T) {
			response, err := strategy.ShouldBid(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, test.should_bid, response.ShouldBid)
		})

		t.Run("ShouldBidBasedOnUsage/"+test.String(), func(t *testing.T) {
			response, err := strategy.ShouldBidBasedOnUsage(context.Background(), request, model.ResourceUsageData{})
			require.NoError(t, err)
			require.Equal(t, test.should_bid, response.ShouldBid)
		})
	}
}
