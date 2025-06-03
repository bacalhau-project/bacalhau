package job

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

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

	historyLong = templates.LongDesc(`
		List job history events for a job by id.
`)

	historyExample = templates.Examples(`
		# All events for a given job.
		bacalhau job history e3f8c209-d683-4a41-b840-f09b88d087b9

		# Job level events
		bacalhau job history --event-type job e3f8c209

		# Execution level events
		bacalhau job history --event-type execution e3f8c209
`)
)

// HistoryOptions is a struct to support node command
type HistoryOptions struct {
	output.OutputOptions
	cliflags.ListOptions
	EventType      string
	ExecutionID    string
	JobVersion     uint64
	AllJobVersions bool
	Namespace      string
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
			api, err := util.NewAPIClientManager(cmd, cfg).GetAuthenticatedAPIClient()
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
	nodeCmd.Flags().Uint64Var(&o.JobVersion, "version", o.JobVersion,
		"The job version to filter by. By default, the latest version is used.")
	nodeCmd.Flags().BoolVar(&o.AllJobVersions, "all-versions", o.AllJobVersions,
		"Specifies that all job versions should be returned. "+
			"By default, only the executions of the latest job version is returned. Note: this flag is mutually "+
			"exclusive with --version, where the latter takes precedence if both are set.")
	nodeCmd.PersistentFlags().StringVar(&o.Namespace, "namespace", o.Namespace,
		`Job Namespace. If not provided, default namespace will be used.`,
	)
	nodeCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	nodeCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return nodeCmd
}

var historyColumns = []output.TableColumn[*models.JobHistory]{
	cols.HistoryDateTime,
	cols.HistoryLevel,
	cols.HistoryJobVersionLevel,
	cols.HistoryExecID,
	cols.HistoryTopic,
	cols.HistoryEvent,
}

func (o *HistoryOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	JobIDOrName := args[0]

	request := &apimodels.ListJobHistoryRequest{
		JobIDOrName: JobIDOrName,
		EventType:   o.EventType,
		ExecutionID: o.ExecutionID,
	}
	request.Limit = o.Limit
	request.NextToken = o.NextToken
	request.OrderBy = o.OrderBy
	request.Reverse = o.Reverse
	request.Namespace = o.Namespace
	request.JobVersion = o.JobVersion
	request.AllJobVersions = o.AllJobVersions

	response, err := api.Jobs().History(ctx, request)
	if err != nil {
		return errors.New(err.Error())
	}

	if err = output.Output(cmd, historyColumns, o.OutputOptions, response.Items); err != nil {
		return fmt.Errorf("failed to output: %w", err)
	}

	return nil
}
