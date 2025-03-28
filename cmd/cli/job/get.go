package job

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

var (
	getLong = templates.LongDesc(`
		Get the results of the job, including stdout and stderr.
`)

	//nolint:lll // Documentation
	getExample = templates.Examples(`
		# Get the results of a job.
		bacalhau job get j-51225160-807e-48b8-88c9-28311c7899e1

		# Get the results of a job, with a short ID.
		bacalhau job get ebd9bf2f
`)
)

type GetOptions struct {
	DownloadSettings *cliflags.DownloaderSettings
}

func NewGetOptions() *GetOptions {
	return &GetOptions{
		DownloadSettings: cliflags.NewDefaultDownloaderSettings(),
	}
}

func NewGetCmd() *cobra.Command {
	OG := NewGetOptions()

	getCmd := &cobra.Command{
		Use:           "get [id]",
		Short:         "Get the results of a job",
		Long:          getLong,
		Example:       getExample,
		Args:          cobra.ExactArgs(1),
		PostRunE:      hook.RemoteCmdPostRunHooks,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
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
			return get(cmd, cmdArgs, api, cfg, OG)
		},
	}

	getCmd.PersistentFlags().AddFlagSet(cliflags.NewDownloadFlags(OG.DownloadSettings))

	return getCmd
}

func get(cmd *cobra.Command, cmdArgs []string, api client.API, cfg types.Bacalhau, OG *GetOptions) error {
	ctx := cmd.Context()

	jobID := cmdArgs[0]
	if jobID == "" {
		byteResult, err := util.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %w", err)
		}
		jobID = string(byteResult)
	}

	// Split the jobID on / to see if the request is for a single file or for the
	// entire jobid.
	parts := strings.SplitN(jobID, "/", 2)
	if len(parts) == 2 {
		jobID, OG.DownloadSettings.SingleFile = parts[0], parts[1]
	}

	if err := util.DownloadResultsHandler(
		ctx,
		cmd,
		cfg,
		api,
		jobID,
		OG.DownloadSettings,
	); err != nil {
		return err
	}

	return nil
}
