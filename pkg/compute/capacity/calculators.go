package capacity

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type DefaultsUsageCalculatorParams struct {
	Defaults model.ResourceUsageData
}

type DefaultsUsageCalculator struct {
	defaults model.ResourceUsageData
}

func NewDefaultsUsageCalculator(params DefaultsUsageCalculatorParams) *DefaultsUsageCalculator {
	return &DefaultsUsageCalculator{
		defaults: params.Defaults,
	}
}

func (c *DefaultsUsageCalculator) Calculate(
	ctx context.Context, job model.Job, parsedUsage model.ResourceUsageData) (model.ResourceUsageData, error) {
	return parsedUsage.Intersect(c.defaults), nil
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
	ctx context.Context, job model.Job, parsedUsage model.ResourceUsageData) (model.ResourceUsageData, error) {
	aggregatedUsage := parsedUsage
	for _, calculator := range c.calculators {
		calculatedUsage, err := calculator.Calculate(ctx, job, parsedUsage)
		if err != nil {
			return model.ResourceUsageData{}, err
		}
		aggregatedUsage = aggregatedUsage.Max(calculatedUsage)
	}
	return aggregatedUsage, nil
}

// Compile-time check to ensure UsageCalculator interface implementation
var _ UsageCalculator = (*DefaultsUsageCalculator)(nil)
var _ UsageCalculator = (*ChainedUsageCalculator)(nil)
