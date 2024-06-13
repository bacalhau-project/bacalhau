//go:build unit || !integration

package semantic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
)

func TestTimeoutStrategy(t *testing.T) {
	tests := []struct {
		name      string
		params    semantic.TimeoutStrategyParams
		request   bidstrategy.BidStrategyRequest
		shouldBid bool
		reason    string
	}{
		{
			name: "client-skip-list",
			params: semantic.TimeoutStrategyParams{
				JobExecutionTimeoutClientIDBypassList: []string{"client"},
			},
			request: bidstrategy.BidStrategyRequest{
				Job: models.Job{
					Namespace: "client",
					Tasks: []*models.Task{
						{
							Timeouts: &models.TimeoutConfig{
								ExecutionTimeout: 9223372036,
							},
						},
					},
				},
			},
			shouldBid: true,
			reason:    "this node does allow client \"client\" to bypass timeout limits",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject := semantic.NewTimeoutStrategy(test.params)

			response, err := subject.ShouldBid(context.Background(), test.request)
			require.NoError(t, err)

			assert.Equal(t, test.shouldBid, response.ShouldBid)
			assert.Equal(t, test.reason, response.Reason)
		})
	}
}
