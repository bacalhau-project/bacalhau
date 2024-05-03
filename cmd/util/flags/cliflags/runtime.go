package cliflags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const DefaultRunWaitSeconds = 600

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
	AutoDownloadResults   bool // Automatically download the results after finishing
}

func RegisterRunTimeFlags(cmd *cobra.Command, s *RunTimeSettings) {
	fs := pflag.NewFlagSet("Runtime settings", pflag.ContinueOnError)
	fs.BoolVar(&s.WaitForJobToFinish, "wait", s.WaitForJobToFinish,
		`Wait for the job to finish. Use --wait=false to return as soon as the job is submitted.`)

	fs.IntVar(&s.WaitForJobTimeoutSecs, "wait-timeout-secs", s.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`)

	fs.BoolVar(&s.PrintJobIDOnly, "id-only", s.PrintJobIDOnly,
		`Print out only the Job ID on successful submission.`)

	fs.BoolVar(&s.PrintNodeDetails, "node-details", s.PrintNodeDetails,
		`Print out details of all nodes (overridden by --id-only).`)

	fs.BoolVarP(&s.Follow, "follow", "f", s.Follow,
		`When specified will follow the output from the job as it runs`)

	fs.BoolVar(&s.DryRun, "dry-run", s.DryRun,
		`Do not submit the job, but instead print out what will be submitted`)

	fs.BoolVar(&s.AutoDownloadResults, "download", s.AutoDownloadResults,
		`Download the job once it completes`)

	cmd.Flags().AddFlagSet(fs)

	cmd.MarkFlagsMutuallyExclusive("dry-run", "wait")
	cmd.MarkFlagsMutuallyExclusive("dry-run", "wait-timeout-secs")
	cmd.MarkFlagsMutuallyExclusive("dry-run", "id-only")
	cmd.MarkFlagsMutuallyExclusive("dry-run", "node-details")
	cmd.MarkFlagsMutuallyExclusive("dry-run", "follow")
	cmd.MarkFlagsMutuallyExclusive("dry-run", "download")

	cmd.MarkFlagsMutuallyExclusive("id-only", "node-details")
	cmd.MarkFlagsMutuallyExclusive("id-only", "follow")
}
