package job

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	describeLong = templates.LongDesc(i18n.T(`
		Full description of a job, in yaml format. 
		Use 'bacalhau job list' to get a list of jobs.
`))
	describeExample = templates.Examples(i18n.T(`
		# Describe a job with the full ID
		bacalhau job describe j-e3f8c209-d683-4a41-b840-f09b88d087b9

		# Describe a job with the a shortened ID
		bacalhau job describe j-47805f5c

		# Describe a job with json output
		bacalhau job describe --output json --pretty j-b6ad164a
`))
)

// DescribeOptions is a struct to support job command
type DescribeOptions struct {
	OutputOpts output.NonTabularOutputOptions
}

// NewDescribeOptions returns initialized Options
func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{
		OutputOpts: output.NonTabularOutputOptions{Format: output.YAMLFormat},
	}
}

func NewDescribeCmd() *cobra.Command {
	o := NewDescribeOptions()
	jobCmd := &cobra.Command{
		Use:     "describe [id]",
		Short:   "Get the info of a job by id.",
		Long:    describeLong,
		Example: describeExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  util.ApplyPorcelainLogLevel,
		Run:     o.run,
	}
	jobCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return jobCmd
}

func (o *DescribeOptions) run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	jobID := args[0]
	response, err := util.GetAPIClientV2(ctx).Jobs().Get(&apimodels.GetJobRequest{
		JobID: jobID,
	})
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("could not get job %s: %w", jobID, err), 1)
	}

	if err = output.OutputOneNonTabular(cmd, o.OutputOpts, response.Job); err != nil {
		util.Fatal(cmd, fmt.Errorf("failed to write job %s: %w", jobID, err), 1)
	}
}
