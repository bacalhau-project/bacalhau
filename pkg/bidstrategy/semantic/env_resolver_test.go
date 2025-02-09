//go:build unit || !integration

package semantic_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute/env"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func TestEnvResolverStrategy(t *testing.T) {
	testCases := []struct {
		name      string
		env       map[string]models.EnvVarValue
		allowList []string
		shouldBid bool
	}{
		{
			name:      "no env vars",
			env:       map[string]models.EnvVarValue{},
			allowList: []string{},
			shouldBid: true,
		},
		{
			name: "literal values only",
			env: map[string]models.EnvVarValue{
				"LITERAL_VAR": "literal-value",
			},
			allowList: []string{},
			shouldBid: true,
		},
		{
			name: "allowed host env var",
			env: map[string]models.EnvVarValue{
				"HOST_VAR": "env:TEST_VAR",
			},
			allowList: []string{"TEST_*"},
			shouldBid: true,
		},
		{
			name: "denied host env var",
			env: map[string]models.EnvVarValue{
				"DENIED_VAR": "env:DENIED_VAR",
			},
			allowList: []string{"TEST_*"},
			shouldBid: false,
		},
		{
			name: "mixed env vars with one denied",
			env: map[string]models.EnvVarValue{
				"LITERAL_VAR": "literal-value",
				"HOST_VAR":    "env:TEST_VAR",
				"DENIED_VAR":  "env:DENIED_VAR",
			},
			allowList: []string{"TEST_*"},
			shouldBid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create resolver with test allowlist
			resolver := env.NewResolver(env.ResolverParams{
				AllowList: tc.allowList,
			})

			// Create strategy with resolver
			strategy := semantic.NewEnvResolverStrategy(semantic.EnvResolverStrategyParams{
				Resolver: resolver,
			})

			// Create job with test env vars
			job := mock.Job()
			job.Task().Env = tc.env

			// Test bid strategy
			response, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
				Job: *job,
			})

			require.NoError(t, err)
			require.Equal(t, tc.shouldBid, response.ShouldBid, fmt.Sprintf("Reason: %s", response.Reason))
		})
	}
}
