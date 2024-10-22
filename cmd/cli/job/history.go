package job

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"k8s.io/kubectl/pkg/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/cols"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

var (
	historyShort = `List history events for a job by id.`

	historyLong = templates.LongDesc(i18n.T(`
		List job history events for a job by id.
`))

	historyExample = templates.Examples(i18n.T(`
		# All events for a given job.
		bacalhau job history e3f8c209-d683-4a41-b840-f09b88d087b9

		# Job level events
		bacalhau job history --type job e3f8c209

		# Execution level events
		bacalhau job history --type execution e3f8c209
`))
)

// HistoryOptions is a struct to support node command
type HistoryOptions struct {
	output.OutputOptions
	cliflags.ListOptions
	EventType   string
	ExecutionID string
}

// NewHistoryOptions returns initialized Options
func NewHistoryOptions() *HistoryOptions {
	return &HistoryOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
		EventType:     "all",
	}
}

func NewHistoryCmd() *cobra.Command {
	o := NewHistoryOptions()
	nodeCmd := &cobra.Command{
		Use:           "history [id]",
		Short:         historyShort,
		Long:          historyLong,
		Example:       historyExample,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.run(cmd, args, api)
		},
	}

	nodeCmd.Flags().StringVar(&o.EventType, "event-type", o.EventType,
		"The type of history events to return. One of: all, job, execution")
	nodeCmd.Flags().StringVar(&o.ExecutionID, "execution-id", o.ExecutionID,
		"The execution id to filter by.")
	nodeCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	nodeCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return nodeCmd
}

var historyColumns = []output.TableColumn[*models.JobHistory]{
	cols.HistoryTime,
	cols.HistoryLevel,
	cols.HistoryExecID,
	cols.HistoryTopic,
	cols.HistoryEvent,
}

func (o *HistoryOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	jobID := args[0]
	response, err := api.Jobs().History(ctx, &apimodels.ListJobHistoryRequest{
		JobID:       jobID,
		EventType:   o.EventType,
		ExecutionID: o.ExecutionID,
		BaseListRequest: apimodels.BaseListRequest{
			Limit:     o.ListOptions.Limit,
			NextToken: o.ListOptions.NextToken,
			OrderBy:   o.ListOptions.OrderBy,
			Reverse:   o.ListOptions.Reverse,
		},
	})
	if err != nil {
		return errors.New(err.Error())
	}

	if err = output.Output(cmd, historyColumns, o.OutputOptions, response.Items); err != nil {
		return fmt.Errorf("failed to output: %w", err)
	}

	return nil
}
