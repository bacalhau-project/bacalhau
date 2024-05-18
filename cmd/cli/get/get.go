package get

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	getLong = templates.LongDesc(i18n.T(`
		Get the results of the job, including stdout and stderr.
`))

	//nolint:lll // Documentation
	getExample = templates.Examples(i18n.T(`
		# Get the results of a job.
		bacalhau get 51225160-807e-48b8-88c9-28311c7899e1

		# Get the results of a job, with a short ID.
		bacalhau get ebd9bf2f
`))
)

type GetOptions struct {
	DownloadSettings *cliflags.DownloaderSettings
}

func NewGetOptions() *GetOptions {
	return &GetOptions{
		DownloadSettings: cliflags.NewDefaultDownloaderSettings(),
	}
}

func NewCmd() *cobra.Command {
	OG := NewGetOptions()

	getFlags := map[string][]configflags.Definition{
		"ipfs": configflags.IPFSFlags,
	}

	getCmd := &cobra.Command{
		Use:      "get [id]",
		Short:    "Get the results of a job",
		Long:     getLong,
		Example:  getExample,
		Args:     cobra.ExactArgs(1),
		PreRunE:  hook.Chain(hook.RemoteCmdPreRunHooks, configflags.PreRun(viper.GetViper(), getFlags)),
		PostRunE: hook.RemoteCmdPostRunHooks,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig()
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return get(cmd, cmdArgs, api, cfg, OG)
		},
	}

	getCmd.PersistentFlags().AddFlagSet(cliflags.NewDownloadFlags(OG.DownloadSettings))

	if err := configflags.RegisterFlags(getCmd, getFlags); err != nil {
		util.Fatal(getCmd, err, 1)
	}

	return getCmd
}

func get(cmd *cobra.Command, cmdArgs []string, api client.API, cfg types.BacalhauConfig, OG *GetOptions) error {
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
		cfg.Node.IPFS,
		api,
		jobID,
		OG.DownloadSettings,
	); err != nil {
		return errors.Wrap(err, "error downloading job")
	}

	return nil
}
