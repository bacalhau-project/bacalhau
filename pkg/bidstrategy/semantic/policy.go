package semantic

import (
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// Create a BidStrategy that implements the passed JobSelectionPolicy.
func FromJobSelectionPolicy(jsp model.JobSelectionPolicy) bidstrategy.SemanticBidStrategy {
	return NewChainedSemanticBidStrategy(
		NewNetworkingStrategy(jsp.AcceptNetworkedJobs),
		NewExternalCommandStrategy(ExternalCommandStrategyParams{
			Command: jsp.ProbeExec,
		}),
		NewExternalHTTPStrategy(ExternalHTTPStrategyParams{
			URL: jsp.ProbeHTTP,
		}),
		NewStatelessJobStrategy(StatelessJobStrategyParams{
			RejectStatelessJobs: jsp.RejectStatelessJobs,
		}),
	)
}
