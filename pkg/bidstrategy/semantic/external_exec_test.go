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
		failMode       bool
		expectedResult bool
	}{
		{
			"fail the response and don't select the job",
			true,
			false,
		},
		{
			"succeed the response and select the job",
			false,
			true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			command := "exit 0"
			if test.failMode {
				command = "exit 1"
			}
			params := semantic.ExternalCommandStrategyParams{
				Command: command,
			}
			strategy := semantic.NewExternalCommandStrategy(params)
			result, err := strategy.ShouldBid(context.Background(), getBidStrategyRequest())
			require.NoError(t, err)
			require.Equal(t, test.expectedResult, result.ShouldBid)
		})
	}
}
