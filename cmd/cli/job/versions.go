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
)

var (
	versionsShort = `List versions for a job`

	versionsLong = templates.LongDesc(`
		List versions for a job by submitting its id or name.
`)

	verionsExample = templates.Examples(`
		# All versions for a given job.
		bacalhau job versions my-job
`)
)

// VersionsOptions is a struct to support node command
type VersionsOptions struct {
	output.OutputOptions
	Namespace string
}

// NewVersionsOptions returns initialized Options
func NewVersionsOptions() *VersionsOptions {
	return &VersionsOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
	}
}

func NewVersionsCmd() *cobra.Command {
	o := NewVersionsOptions()

	jobVersionsCmd := &cobra.Command{
		Use:           "versions",
		Short:         versionsShort,
		Long:          versionsLong,
		Example:       verionsExample,
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

	jobVersionsCmd.SilenceUsage = true
	jobVersionsCmd.SilenceErrors = true

	jobVersionsCmd.PersistentFlags().StringVar(&o.Namespace, "namespace", o.Namespace,
		`Job Namespace. If not provided, default namespace will be used.`,
	)
	jobVersionsCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return jobVersionsCmd
}

var (
	columnJobCreatedAt = output.TableColumn[*models.Job]{
		ColumnConfig: table.ColumnConfig{Name: "Created At", WidthMax: 30, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.Job) string { return j.GetModifyTime().Format(time.DateTime) },
	}
	columnJobVersionsNumber = output.TableColumn[*models.Job]{
		ColumnConfig: table.ColumnConfig{Name: "Job Version", WidthMax: 11, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.Job) string { return strconv.FormatUint(j.Version, 10) },
	}
)

var versionsColumns = []output.TableColumn[*models.Job]{
	columnJobCreatedAt,
	columnJobVersionsNumber,
}

func (o *VersionsOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	jobIDOrName := args[0]
	request := &apimodels.ListJobVersionsRequest{
		JobIDOrName: jobIDOrName,
	}

	request.Namespace = o.Namespace

	response, err := api.Jobs().Versions(ctx, request)
	if err != nil {
		return errors.New(err.Error())
	}

	if err = output.Output(cmd, versionsColumns, o.OutputOptions, response.Items); err != nil {
		return fmt.Errorf("failed to output: %w", err)
	}

	return nil
}
