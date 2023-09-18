package cliflags

import "github.com/spf13/pflag"

func DefaultRunTimeSettings() *RunTimeSettings {
	return &RunTimeSettings{
		WaitForJobToFinish:    true,
		WaitForJobTimeoutSecs: DefaultRunWaitSeconds,
		PrintJobIDOnly:        false,
		Follow:                false,
		DryRun:                false,
	}
}

type RunTimeSettings struct {
	WaitForJobToFinish    bool // Wait for the job to finish before returning
	WaitForJobTimeoutSecs int  // Timeout for waiting for the job to finish
	PrintJobIDOnly        bool // Only print the Job ID as output
	PrintNodeDetails      bool
	Follow                bool // Follow along with the output of the job
	DryRun                bool // iff true do not submit the job, but instead print out what will be submitted.
}

const DefaultRunWaitSeconds = 600

func NewRunTimeSettingsFlags(settings *RunTimeSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Runtime settings", pflag.ContinueOnError)
	flags.BoolVar(&settings.WaitForJobToFinish, "wait", settings.WaitForJobToFinish,
		`Wait for the job to finish.`)
	flags.IntVar(&settings.WaitForJobTimeoutSecs, "wait-timeout-secs", settings.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`)
	flags.BoolVar(&settings.PrintJobIDOnly, "id-only", settings.PrintJobIDOnly,
		`Print out only the Job ID on successful submission.`)
	flags.BoolVar(&settings.PrintNodeDetails, "node-details", settings.PrintNodeDetails,
		`Print out details of all nodes (overridden by --id-only).`)
	flags.BoolVarP(&settings.Follow, "follow", "f", settings.Follow,
		`When specified will follow the output from the job as it runs`)
	flags.BoolVar(
		&settings.DryRun, "dry-run", settings.DryRun,
		`Do not submit the job, but instead print out what will be submitted`)

	return flags
}
