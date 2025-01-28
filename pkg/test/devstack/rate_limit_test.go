//go:build integration || !unit

package devstack

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type RateLimitSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackRateLimitSuite(t *testing.T) {
	suite.Run(t, new(RateLimitSuite))
}

func (s *RateLimitSuite) TestJobRateLimit() {
	tests := []struct {
		name      string
		jobType   string
		nodeCount int
	}{
		{
			name:      "batch job with count greater than rate limit",
			jobType:   models.JobTypeBatch,
			nodeCount: 6,
		},
		{
			name:      "ops job with nodes greater than rate limit",
			jobType:   models.JobTypeOps,
			nodeCount: 6,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			job := &models.Job{
				Name: s.T().Name(),
				Type: tt.jobType,
				Tasks: []*models.Task{
					{
						Name: s.T().Name(),
						Engine: &models.SpecConfig{
							Type:   models.EngineNoop,
							Params: make(map[string]interface{}),
						},
						Publisher: publisher_local.NewSpecConfig(),
					},
				},
			}

			if tt.jobType == models.JobTypeBatch {
				job.Count = tt.nodeCount
			}

			testCase := scenario.Scenario{
				Stack: &scenario.StackConfig{
					DevStackOptions: []devstack.ConfigOption{
						devstack.WithNumberOfRequesterOnlyNodes(1),
						devstack.WithNumberOfComputeOnlyNodes(tt.nodeCount),
						devstack.WithSystemConfig(node.SystemConfig{
							MaxExecutionsPerEval:  2, // Small limit to test rate limiting
							ExecutionLimitBackoff: 50 * time.Millisecond,
						}),
					},
				},
				Job: job,
				JobCheckers: []scenario.StateChecks{
					scenario.WaitForSuccessfulCompletion(),
					scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
						models.ExecutionStateCompleted: tt.nodeCount,
					}),
				},
			}

			s.RunScenario(testCase)
		})
	}
}
