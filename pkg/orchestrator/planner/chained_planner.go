package planner

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// ChainedPlanner implements the orchestrator.Planner interface by chaining multiple planners together.
type ChainedPlanner struct {
	Planners []orchestrator.Planner
}

// NewChain creates a new ChainedPlanner with the provided planners.
func NewChain(planners ...orchestrator.Planner) *ChainedPlanner {
	return &ChainedPlanner{
		Planners: planners,
	}
}

// Add adds the specified planners to the ChainedPlanner.
func (c *ChainedPlanner) Add(planners ...orchestrator.Planner) {
	c.Planners = append(c.Planners, planners...)
}

// Process executes the planning process by invoking each planner in the chain.
func (c *ChainedPlanner) Process(ctx context.Context, plan *models.Plan) error {
	for _, p := range c.Planners {
		err := p.Process(ctx, plan)
		if err != nil {
			return fmt.Errorf("failed to process plan with planner %T: %w", p, err)
		}
	}
	return nil
}

// compile-time check whether the ChainedPlanner implements the Planner interface.
var _ orchestrator.Planner = (*ChainedPlanner)(nil)
