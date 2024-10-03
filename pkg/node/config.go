package node

import (
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// SystemConfig is node configuration that cannot be specified by the user.
// They are meant for internal use only and to override the node's behaviour for testing purposes
type SystemConfig struct {
	///////////////////////////////
	// Orchestrator Specific Config
	///////////////////////////////

	// RetryStrategy overrides the orchestrator's retry strategy for testing purposes
	RetryStrategy orchestrator.RetryStrategy

	// OverSubscriptionFactor overrides the orchestrator's over subscription factor for testing purposes
	OverSubscriptionFactor float64

	// NodeRankRandomnessRange overrides the node's rank randomness range for testing purposes
	NodeRankRandomnessRange int

	///////////////////////////////
	// Compute Specific Config
	///////////////////////////////

	// BidSemanticStrategy overrides the node's bid semantic strategy for testing purposes
	BidSemanticStrategy bidstrategy.BidStrategy
	// BidResourceStrategy overrides the node's bid resource strategy for testing purposes
	BidResourceStrategy bidstrategy.BidStrategy

	// TODO: remove compute level resource defaults. This should be handled at the orchestrator,
	//  but we still need to validate the behaviour is a job without resource limits land on a compute node
	DefaultComputeJobResourceLimits models.Resources
}

func DefaultSystemConfig() SystemConfig {
	return SystemConfig{
		OverSubscriptionFactor:  1.5,
		NodeRankRandomnessRange: 5,
		DefaultComputeJobResourceLimits: models.Resources{
			CPU:    0.1,               // 100m
			Memory: 100 * 1024 * 1024, // 100Mi
		},
	}
}

// applyDefaults applies the default values to the system config
func (c *SystemConfig) applyDefaults() {
	defaults := DefaultSystemConfig()
	if c.OverSubscriptionFactor == 0 {
		c.OverSubscriptionFactor = defaults.OverSubscriptionFactor
	}
	if c.NodeRankRandomnessRange == 0 {
		c.NodeRankRandomnessRange = defaults.NodeRankRandomnessRange
	}
	if c.DefaultComputeJobResourceLimits.IsZero() {
		c.DefaultComputeJobResourceLimits = defaults.DefaultComputeJobResourceLimits
	}
}
