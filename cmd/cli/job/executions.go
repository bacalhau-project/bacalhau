package job

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

var executionsOrderByFields = []string{"modified_at", "created_at"}

var (
	executionShort = `List executions for a job by id.`

	executionLong = templates.LongDesc(`
		List executions for a job by id.
`)

	executionExample = templates.Examples(`
		# All executions for a given job.
		bacalhau job executions j-e3f8c209-d683-4a41-b840-f09b88d087b9
`)
)

// ExecutionOptions is a struct to support node command
type ExecutionOptions struct {
	output.OutputOptions
	cliflags.ListOptions
}

// NewExecutionOptions returns initialized Options
func NewExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
		ListOptions: cliflags.ListOptions{
			Limit:         20,
			OrderByFields: executionsOrderByFields,
		},
	}
}

func NewExecutionCmd() *cobra.Command {
	o := NewExecutionOptions()

	nodeCmd := &cobra.Command{
		Use:           "executions [id]",
		Short:         executionShort,
		Long:          executionLong,
		Example:       executionExample,
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

	nodeCmd.SilenceUsage = true
	nodeCmd.SilenceErrors = true

	nodeCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	nodeCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return nodeCmd
}

var (
	executionColumnCreated = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{Name: "Created", WidthMax: 8, WidthMaxEnforcer: output.ShortenTime},
		Value:        func(e *models.Execution) string { return e.GetCreateTime().Format(time.DateTime) },
	}
	executionColumnModified = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{Name: "Modified", WidthMax: 8, WidthMaxEnforcer: output.ShortenTime},
		Value:        func(e *models.Execution) string { return e.GetModifyTime().Format(time.DateTime) },
	}
	executionColumnCreatedSince = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{Name: "Created", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(e *models.Execution) string { return output.Elapsed(e.GetCreateTime()) },
	}
	executionColumnModifiedSince = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{Name: "Modified", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(e *models.Execution) string { return output.Elapsed(e.GetModifyTime()) },
	}
	executionColumnID = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{
			Name:             "ID",
			WidthMax:         idgen.ShortIDLengthWithPrefix,
			WidthMaxEnforcer: func(col string, maxLen int) string { return idgen.ShortUUID(col) }},
		Value: func(e *models.Execution) string { return e.ID },
	}
	executionColumnNodeID = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{
			Name:             "Node ID",
			WidthMax:         idgen.ShortIDLengthWithPrefix,
			WidthMaxEnforcer: func(col string, maxLen int) string { return idgen.ShortUUID(col) }},
		Value: func(e *models.Execution) string { return e.NodeID },
	}
	executionColumnRev = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{Name: "Rev.", WidthMax: 4, WidthMaxEnforcer: text.WrapText},
		Value:        func(e *models.Execution) string { return strconv.FormatUint(e.Revision, 10) },
	}
	executionColumnState = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{Name: "State", WidthMax: 17, WidthMaxEnforcer: text.WrapText},
		Value:        func(e *models.Execution) string { return e.ComputeState.StateType.String() },
	}
	executionColumnDesired = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{Name: "Desired", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(e *models.Execution) string { return e.DesiredState.StateType.String() },
	}
	executionColumnComment = output.TableColumn[*models.Execution]{
		ColumnConfig: table.ColumnConfig{
			Name: "Comment", WidthMax: 40, WidthMaxEnforcer: output.WrapSoftPreserveNewlines},
		Value: func(e *models.Execution) string { return e.ComputeState.Message },
	}
)

var executionColumns = []output.TableColumn[*models.Execution]{
	executionColumnCreated,
	executionColumnModified,
	executionColumnID,
	executionColumnNodeID,
	executionColumnRev,
	executionColumnState,
	executionColumnDesired,
}

func (o *ExecutionOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	jobID := args[0]
	response, err := api.Jobs().Executions(ctx, &apimodels.ListJobExecutionsRequest{
		JobID: jobID,
		BaseListRequest: apimodels.BaseListRequest{
			Limit:     o.Limit,
			NextToken: o.NextToken,
			OrderBy:   o.OrderBy,
			Reverse:   o.Reverse,
		},
	})
	if err != nil {
		return errors.New(err.Error())
	}

	if err = output.Output(cmd, executionColumns, o.OutputOptions, response.Items); err != nil {
		return fmt.Errorf("failed to output: %w", err)
	}

	return nil
}
