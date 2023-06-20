package get

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/handler"
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
	IPFSDownloadSettings *flags.DownloaderSettings
}

func NewGetOptions() *GetOptions {
	return &GetOptions{
		IPFSDownloadSettings: flags.NewDefaultDownloadSettings(),
	}
}

func NewCmd() *cobra.Command {
	OG := NewGetOptions()

	getCmd := &cobra.Command{
		Use:     "get [id]",
		Short:   "Get the results of a job",
		Long:    getLong,
		Example: getExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  handler.ApplyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return get(cmd, cmdArgs, OG)
		},
	}

	getCmd.PersistentFlags().AddFlagSet(flags.NewDownloadFlags(OG.IPFSDownloadSettings))

	return getCmd
}

func get(cmd *cobra.Command, cmdArgs []string, OG *GetOptions) error {
	ctx := cmd.Context()

	jobID := cmdArgs[0]
	if jobID == "" {
		byteResult, err := handler.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %w", err)
		}
		jobID = string(byteResult)
	}

	// Split the jobID on / to see if the request is for a single file or for the
	// entire jobid.
	parts := strings.SplitN(jobID, "/", 2)
	if len(parts) == 2 {
		jobID, OG.IPFSDownloadSettings.SingleFile = parts[0], parts[1]
	}

	err := handler.DownloadResultsHandler(
		ctx,
		cmd,
		jobID,
		*OG.IPFSDownloadSettings,
	)

	if err != nil {
		return errors.Wrap(err, "error downloading job")
	}

	return nil
}
