//go:build unit || !integration

package semantic_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
)

func TestJobSelectionExec(t *testing.T) {
	testCases := []struct {
		name           string
		testCommand    string
		expectedResult bool
		expectedReason string
	}{
		{
			"fail the response and don't select the job",
			"exit 1",
			false,
			"this node does not accept jobs where external command \"exit 1\" returns exit code 1",
		},
		{
			"succeed the response and select the job",
			"exit 0",
			true,
			"this node does accept jobs where external command \"exit 0\" returns exit code 0",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			params := semantic.ExternalCommandStrategyParams{Command: test.testCommand}
			strategy := semantic.NewExternalCommandStrategy(params)
			result, err := strategy.ShouldBid(context.Background(), getBidStrategyRequest(t))
			require.NoError(t, err)
			require.Equal(t, test.expectedResult, result.ShouldBid)
			require.Equal(t, test.expectedReason, result.Reason)
		})
	}
}
