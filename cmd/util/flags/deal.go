package flags

import (
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type DealFlagSettings struct {
	TargetingMode model.TargetingMode
	Concurrency   int // Number of concurrent jobs to run
	Confidence    int // Minimum number of nodes that must agree on a verification result
	MinBids       int // Minimum number of bids before they will be accepted (at random)
}

func DealFlags(settings *DealFlagSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Deal settings", pflag.ContinueOnError)
	flags.IntVar(
		&settings.Concurrency,
		"concurrency",
		settings.Concurrency,
		`How many nodes should run the job`,
	)
	flags.IntVar(
		&settings.Confidence, "confidence", settings.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
	)
	flags.IntVar(
		&settings.MinBids,
		"min-bids",
		settings.MinBids,
		`Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)`,
	)
	flags.Var(TargetingFlag(&settings.TargetingMode), "target",
		`Whether to target the minimum number of matching nodes ("any") (default) or all matching nodes ("all")`)
	return flags
}
