package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/downloader/util"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
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
	IPFSDownloadSettings *model.DownloaderSettings
}

func NewGetOptions() *GetOptions {
	return &GetOptions{
		IPFSDownloadSettings: util.NewDownloadSettings(),
	}
}

func newGetCmd() *cobra.Command {
	OG := NewGetOptions()

	getCmd := &cobra.Command{
		Use:     "get [id]",
		Short:   "Get the results of a job",
		Long:    getLong,
		Example: getExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return get(cmd, cmdArgs, OG)
		},
	}

	getCmd.PersistentFlags().AddFlagSet(NewIPFSDownloadFlags(OG.IPFSDownloadSettings))

	return getCmd
}

func get(cmd *cobra.Command, cmdArgs []string, OG *GetOptions) error {
	ctx := cmd.Context()

	cm := cmd.Context().Value(systemManagerKey).(*system.CleanupManager)

	var err error

	jobID := cmdArgs[0]
	if jobID == "" {
		var byteResult []byte
		byteResult, err = ReadFromStdinIfAvailable(cmd, cmdArgs)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Unknown error reading from file: %s\n", err), 1)
			return err
		}
		jobID = string(byteResult)
	}

	err = downloadResultsHandler(
		ctx,
		cm,
		cmd,
		jobID,
		*OG.IPFSDownloadSettings,
	)

	if err != nil {
		return errors.Wrap(err, "error downloading job")
	}

	return nil
}
