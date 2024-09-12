package job

import (
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	validateLong = templates.LongDesc(i18n.T(`
		Validate a job from a file
		JSON and YAML formats are accepted.
		Job Specification: https://docs.bacalhau.org/setting-up/jobs/job
`))

	//nolint:lll // Documentation
	validateExample = templates.Examples(i18n.T(`
		# Validate a job using the data in job.yaml
		bacalhau job validate ./job.yaml
`))
)

const JobSpecLink = "https://docs.bacalhau.org/setting-up/jobs/job"

func NewValidateCmd() *cobra.Command {
	validateCmd := &cobra.Command{
		Use:     "validate",
		Short:   "validate a job using a json or yaml file.",
		Long:    validateLong,
		Example: validateExample,
		Args:    cobra.MinimumNArgs(1),
		// so we don't print the usage when a job is invalid, just print the validation errors
		// --help will still show usage
		SilenceUsage: true,
		// so we don't print the error twice
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			err := run(cmd, cmdArgs)
			if err != nil {
				cmd.Println("Error: " + userstrings.JobSpecBad)
				cmd.Println("Job Specification: " + JobSpecLink)
				cmd.Println()
				return err
			}
			cmd.Println("OK")
			return nil
		},
	}

	validateCmd.SilenceUsage = true
	validateCmd.SilenceErrors = true

	return validateCmd
}

func run(cmd *cobra.Command, args []string) error {
	// read the job spec from stdin or file
	jobBytes, err := util.ReadJobFromUser(cmd, args)
	if err != nil {
		return err
	}

	j, err := marshaller.UnmarshalJob(jobBytes)
	if err != nil {
		return err
	}

	// Normalize and validate the job spec
	j.Normalize()
	if err := j.ValidateSubmission(); err != nil {
		return err
	}

	if warnings := j.SanitizeSubmission(); len(warnings) > 0 {
		for _, w := range warnings {
			cmd.Println(w)
		}
	}

	return nil
}
