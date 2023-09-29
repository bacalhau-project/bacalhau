package capacity

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type DefaultsUsageCalculatorParams struct {
	Defaults models.Resources
}

type DefaultsUsageCalculator struct {
	defaults models.Resources
}

func NewDefaultsUsageCalculator(params DefaultsUsageCalculatorParams) *DefaultsUsageCalculator {
	return &DefaultsUsageCalculator{
		defaults: params.Defaults,
	}
}

func (c *DefaultsUsageCalculator) Calculate(
	ctx context.Context, job models.Job, parsedUsage models.Resources) (*models.Resources, error) {
	return parsedUsage.Merge(c.defaults), nil
}

type ChainedUsageCalculatorParams struct {
	Calculators []UsageCalculator
}

type ChainedUsageCalculator struct {
	calculators []UsageCalculator
}

func NewChainedUsageCalculator(params ChainedUsageCalculatorParams) *ChainedUsageCalculator {
	return &ChainedUsageCalculator{
		calculators: params.Calculators,
	}
}

func (c *ChainedUsageCalculator) Calculate(
	ctx context.Context, job models.Job, parsedUsage models.Resources) (*models.Resources, error) {
	aggregatedUsage := &parsedUsage
	for _, calculator := range c.calculators {
		calculatedUsage, err := calculator.Calculate(ctx, job, parsedUsage)
		if err != nil {
			return nil, err
		}
		aggregatedUsage = aggregatedUsage.Max(*calculatedUsage)
	}
	return aggregatedUsage, nil
}

// Compile-time check to ensure UsageCalculator interface implementation
var _ UsageCalculator = (*DefaultsUsageCalculator)(nil)
var _ UsageCalculator = (*ChainedUsageCalculator)(nil)
