//go:build unit || !integration

package semantic_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
)

type networkingStrategyTestCase struct {
	accept         bool
	job_networking models.NetworkConfig
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
	{false, models.NetworkConfig{Type: models.NetworkNone}, true},
	{false, models.NetworkConfig{Type: models.NetworkFull}, false},
	{true, models.NetworkConfig{Type: models.NetworkNone}, true},
	{true, models.NetworkConfig{Type: models.NetworkFull}, true},
}

func TestNetworkingStrategy(t *testing.T) {
	for _, test := range networkingStrategyTestCases {
		job := mock.Job()
		job.Task().Network = &test.job_networking
		strategy := semantic.NewNetworkingStrategy(test.accept)
		request := bidstrategy.BidStrategyRequest{Job: *job}

		t.Run("ShouldBid/"+test.String(), func(t *testing.T) {
			response, err := strategy.ShouldBid(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, test.should_bid, response.ShouldBid)
		})

	}
}
