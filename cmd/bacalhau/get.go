package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/userstrings"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
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
			OutputDir:      ".",
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
			// If there's no input ond no stdin, then cmdArgs is nil, and byteResult is nil.
			if err.Error() == userstrings.NoStdInProvidedErrorString || byteResult == nil {
				// Both filename and stdin are empty
				Fatal(userstrings.NoFilenameProvidedErrorString, 1)
			} else if err != nil {
				// Error not related to fields being empty
				return err
			}
			jobID = string(byteResult)
		}

		cmd.Printf("Fetching results of job '%s'...", jobID)

		j, _, err := GetAPIClient().Get(ctx, jobID)

		if err != nil {
			if _, ok := err.(*bacerrors.JobNotFound); ok {
				cmd.Printf("job not found.\n")
				Fatal("", 1)
			} else {
				Fatal(fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", jobID, err), 1)
			}
		}

		results, err := GetAPIClient().GetResults(ctx, j.ID)
		if err != nil {
			Fatal(fmt.Sprintf("Error getting results for job ID (%s): %s", jobID, err), 1)
		}

		err = ipfs.DownloadJob(
			ctx,
			cm,
			j,
			results,
			OG.IPFSDownloadSettings,
		)

		if err != nil {
			Fatal(fmt.Sprintf("Error downloading results from job ID (%s): %s", jobID, err), 1)
		}

		return nil
	},
}
