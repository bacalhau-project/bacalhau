package flags

import (
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func NewDefaultDealFlagSettings() *DealFlagSettings {
	return &DealFlagSettings{
		Concurrency:   1,
		TargetingMode: model.TargetAny,
	}
}

type DealFlagSettings struct {
	TargetingMode model.TargetingMode
	Concurrency   int // Number of concurrent jobs to run
}

func DealFlags(settings *DealFlagSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Deal settings", pflag.ContinueOnError)
	flags.IntVar(
		&settings.Concurrency,
		"concurrency",
		settings.Concurrency,
		`How many nodes should run the job`,
	)
	flags.Var(TargetingFlag(&settings.TargetingMode), "target",
		`Whether to target the minimum number of matching nodes ("any") (default) or all matching nodes ("all")`)
	return flags
}
