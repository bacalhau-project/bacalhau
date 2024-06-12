package job

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	schema_job "github.com/bacalhau-project/bacalhau/schemas/job"
)

var (
	validateLong = templates.LongDesc(i18n.T(`
		Validate a job from a file

		JSON and YAML formats are accepted.
`))

	//nolint:lll // Documentation
	validateExample = templates.Examples(i18n.T(`
		# Validate a job using the data in job.yaml
		bacalhau job validate ./job.yaml
`))
)

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
			return run(cmd, cmdArgs)
		},
	}
	return validateCmd
}

func run(cmd *cobra.Command, cmdArgs []string) error {
	if len(cmdArgs) == 0 {
		return fmt.Errorf("you must specify a filename or provide the content to be validated via stdin")
	}

	filePath := cmdArgs[0]
	var result *gojsonschema.Result
	var err error

	schema, err := schema_job.Schema()
	if err != nil {
		return err
	}
	result, err = schema.ValidateFile(filePath)
	if err != nil {
		return fmt.Errorf("running validation: %w", err)
	}

	if result.Valid() {
		cmd.Println("The Job is valid")
	} else {
		msg := "The Job is not valid. See errors:\n"
		for _, desc := range result.Errors() {
			msg += fmt.Sprintf("- %s\n", desc)
		}
		return fmt.Errorf(msg)
	}
	return nil
}
