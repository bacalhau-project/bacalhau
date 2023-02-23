package bidstrategy

import "github.com/filecoin-project/bacalhau/pkg/model"

// Create a BidStrategy that implements the passed JobSelectionPolicy.
func FromJobSelectionPolicy(jsp model.JobSelectionPolicy) BidStrategy {
	return NewChainedBidStrategy(
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
