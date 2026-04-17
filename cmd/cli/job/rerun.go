package job

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
)

type RerunOptions struct {
	RunTimeSettings *cliflags.RunTimeSettings // Run time settings for execution (e.g. follow, wait after submission)
	JobVersion      uint64
}

func NewRerunOptions() *RerunOptions {
	return &RerunOptions{
		RunTimeSettings: cliflags.DefaultRunTimeSettings(),
	}
}

func NewRerunCmd() *cobra.Command {
	o := NewRerunOptions()

	rerunCmd := &cobra.Command{
		Use:           "rerun",
		Short:         "Rerun a job using its id or name.",
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
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

	rerunCmd.Flags().VarP(util.NewUintValue(0, &o.JobVersion), "version", "v",
		"The job version to rerun. Defaults to latest version.")
	rerunCmd.Flags().AddFlagSet(cliflags.NewRunTimeSettingsFlagsWithoutDryRun(o.RunTimeSettings))

	return rerunCmd
}

func (o *RerunOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	jobIDOrName := args[0]
	resp, err := api.Jobs().Rerun(ctx, &apimodels.RerunJobRequest{
		JobIDOrName: jobIDOrName,
		JobVersion:  o.JobVersion,
	})
	if err != nil {
		return fmt.Errorf("failed API request: %w", err)
	}

	jobProgressJobModel := &models.Job{
		ID:      resp.JobID,
		Version: resp.JobVersion,
	}
	jobProgressPrinter := printer.NewJobProgressPrinter(api, o.RunTimeSettings)
	if err := jobProgressPrinter.PrintJobProgress(ctx, jobProgressJobModel, cmd); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}
