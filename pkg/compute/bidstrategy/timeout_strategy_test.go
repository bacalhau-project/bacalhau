//go:build unit || !integration

package bidstrategy

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutStrategy(t *testing.T) {
	tests := []struct {
		name      string
		params    TimeoutStrategyParams
		request   BidStrategyRequest
		shouldBid bool
		reason    string
	}{
		{
			name: "timeout-too-large",
			params: TimeoutStrategyParams{
				JobExecutionTimeoutClientIDBypassList: []string{"client"},
			},
			request: BidStrategyRequest{
				Job: model.Job{
					Metadata: model.Metadata{ClientID: "client"},
					Spec:     model.Spec{Timeout: 9223372036.1},
				},
			},
			shouldBid: false,
			reason:    "job timeout 9223372036.1 exceeds maximum possible value",
		},
		{
			name: "client-skip-list",
			params: TimeoutStrategyParams{
				JobExecutionTimeoutClientIDBypassList: []string{"client"},
			},
			request: BidStrategyRequest{
				Job: model.Job{
					Metadata: model.Metadata{ClientID: "client"},
					Spec:     model.Spec{Timeout: 9223372036},
				},
			},
			shouldBid: true,
			reason:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject := NewTimeoutStrategy(test.params)

			response, err := subject.ShouldBid(context.Background(), test.request)
			require.NoError(t, err)

			assert.Equal(t, test.shouldBid, response.ShouldBid)
			assert.Equal(t, test.reason, response.Reason)
		})
	}
}
