package cliflags

import "github.com/spf13/pflag"

func NewDefaultRunTimeSettings() *RunTimeSettings {
	return &RunTimeSettings{
		AutoDownloadResults:   false,
		WaitForJobToFinish:    true,
		WaitForJobTimeoutSecs: DefaultRunWaitSeconds,
		IPFSGetTimeOut:        10,
		IsLocal:               false,
		PrintJobIDOnly:        false,
		PrintNodeDetails:      false,
		Follow:                false,
		SkipSyntaxChecking:    false,
		DryRun:                false,
	}
}

type RunTimeSettings struct {
	AutoDownloadResults   bool // Automatically download the results after finishing
	IPFSGetTimeOut        int  // Timeout for IPFS in seconds
	IsLocal               bool // Job should be executed locally
	WaitForJobToFinish    bool // Wait for the job to finish before returning
	WaitForJobTimeoutSecs int  // Timeout for waiting for the job to finish
	PrintJobIDOnly        bool // Only print the Job ID as output
	PrintNodeDetails      bool // Print the node details as output
	Follow                bool // Follow along with the output of the job
	SkipSyntaxChecking    bool // Skip having 'shellchecker' verify syntax of the command.
	DryRun                bool // iff true do not submit the job, but instead print out what will be submitted.
}

const DefaultRunWaitSeconds = 600

func NewRunTimeSettingsFlags(settings *RunTimeSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Runtime settings", pflag.ContinueOnError)
	flags.IntVarP(&settings.IPFSGetTimeOut, "gettimeout", "g", settings.IPFSGetTimeOut,
		`Timeout for getting the results of a job in --wait`)
	flags.BoolVar(&settings.IsLocal, "local", settings.IsLocal,
		`Run the job locally. Docker is required`)
	flags.BoolVar(&settings.WaitForJobToFinish, "wait", settings.WaitForJobToFinish,
		`Wait for the job to finish.`)
	flags.IntVar(&settings.WaitForJobTimeoutSecs, "wait-timeout-secs", settings.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`)
	flags.BoolVar(&settings.PrintJobIDOnly, "id-only", settings.PrintJobIDOnly,
		`Print out only the Job ID on successful submission.`)
	flags.BoolVar(&settings.PrintNodeDetails, "node-details", settings.PrintNodeDetails,
		`Print out details of all nodes (overridden by --id-only).`)
	flags.BoolVar(&settings.AutoDownloadResults, "download", settings.AutoDownloadResults,
		`Should we download the results once the job is complete?`)
	flags.BoolVarP(&settings.Follow, "follow", "f", settings.Follow,
		`When specified will follow the output from the job as it runs`)
	flags.BoolVar(
		&settings.SkipSyntaxChecking, "skip-syntax-checking", settings.SkipSyntaxChecking,
		`Skip having 'shellchecker' verify syntax of the command`)
	flags.BoolVar(
		&settings.DryRun, "dry-run", settings.DryRun,
		`Do not submit the job, but instead print out what will be submitted`)

	return flags
}
