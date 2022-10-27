package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
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

	// Set Defaults (probably a better way to do this)
	OG = NewGetOptions()

	// For the -f flag
)

type GetOptions struct {
	IPFSDownloadSettings ipfs.IPFSDownloadSettings
}

func NewGetOptions() *GetOptions {
	return &GetOptions{
		IPFSDownloadSettings: ipfs.IPFSDownloadSettings{
			TimeoutSecs:    int(ipfs.DefaultIPFSTimeout.Seconds()),
			OutputDir:      "",
			IPFSSwarmAddrs: "",
		},
	}
}

func init() { //nolint:gochecknoinits
	setupDownloadFlags(getCmd, &OG.IPFSDownloadSettings)
}

var getCmd = &cobra.Command{
	Use:     "get [id]",
	Short:   "Get the results of a job",
	Long:    getLong,
	Example: getExample,
	Args:    cobra.ExactArgs(1),
	PreRun:  applyPorcelainLogLevel,
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, span := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/get")
		defer span.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		var err error

		jobID := cmdArgs[0]
		if jobID == "" {
			var byteResult []byte
			byteResult, err = ReadFromStdinIfAvailable(cmd, cmdArgs)
			if err != nil {
				Fatal(fmt.Sprintf("Unknown error reading from file: %s\n", err), 1)
				return err
			}
			jobID = string(byteResult)
		}

		err = downloadResultsHandler(
			ctx,
			cm,
			cmd,
			jobID,
			OG.IPFSDownloadSettings,
		)

		if err != nil {
			return errors.Wrap(err, "error downloading job")
		}

		return nil
	},
}
