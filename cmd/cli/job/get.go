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

	getExample = templates.Examples(`
		# Get the results of a job using its Name.
		bacalhau job get my-job

		# Get the results of a job using its ID.
		bacalhau job get j-51225160-807e-48b8-88c9-28311c7899e1

		# Get the results of a job, with a short ID.
		bacalhau job get ebd9bf2f
`)
)

type GetOptions struct {
	Namespace        string
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
		Use:           "get",
		Short:         "Get the results of a job by ID or Name",
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

	getCmd.PersistentFlags().StringVar(&OG.Namespace, "namespace", OG.Namespace,
		`Job Namespace. If not provided, default namespace will be used.`,
	)
	getCmd.PersistentFlags().AddFlagSet(cliflags.NewDownloadFlags(OG.DownloadSettings))

	return getCmd
}

func get(cmd *cobra.Command, cmdArgs []string, api client.API, cfg types.Bacalhau, OG *GetOptions) error {
	ctx := cmd.Context()

	jobIDOrName := cmdArgs[0]
	if jobIDOrName == "" {
		byteResult, err := util.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %w", err)
		}
		jobIDOrName = string(byteResult)
	}

	// Split the jobIDOrName on / to see if the request is for a single file or for the
	// entire jobid.
	// TODO: Enforce certain syntax for JobName - only DNS compatible names should be allowed
	parts := strings.SplitN(jobIDOrName, "/", 2)
	if len(parts) == 2 {
		jobIDOrName, OG.DownloadSettings.SingleFile = parts[0], parts[1]
	}

	if err := util.DownloadResultsHandler(
		ctx,
		cmd,
		cfg,
		api,
		jobIDOrName,
		OG.Namespace,
		OG.DownloadSettings,
	); err != nil {
		return err
	}

	return nil
}
