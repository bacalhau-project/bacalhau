package bacalhau

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
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

func getDefaultJobFolder(jobID string) string {
	return fmt.Sprintf("job-%s", system.GetShortID(jobID))
}

// if the user does not supply a value for "download results to here"
// then we default to making a folder in the current directory
func ensureDefaultDownloadLocation(jobID string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	downloadDir := filepath.Join(cwd, getDefaultJobFolder(jobID))
	err = os.MkdirAll(downloadDir, os.ModeDir)
	if err != nil {
		return "", err
	}
	return downloadDir, nil
}

func processDownloadSettings(settings ipfs.IPFSDownloadSettings, jobID string) (ipfs.IPFSDownloadSettings, error) {
	if settings.OutputDir == "" {
		dir, err := ensureDefaultDownloadLocation(jobID)
		if err != nil {
			return settings, err
		}
		settings.OutputDir = dir
	}
	return settings, nil
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

		fmt.Fprintf(os.Stderr, "Fetching results of job '%s'...\n", jobID)

		j, _, err := GetAPIClient().Get(ctx, jobID)

		if err != nil {
			if _, ok := err.(*bacerrors.JobNotFound); ok {
				cmd.Printf("job not found.\n")
				Fatal("", 1)
			} else {
				Fatal(fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", jobID, err), 1)
			}
			return err
		}

		results, err := GetAPIClient().GetResults(ctx, j.ID)
		if err != nil {
			Fatal(fmt.Sprintf("Error getting results for job ID (%s): %s", jobID, err), 1)
			return err
		}

		processedDownloadSettings, err := processDownloadSettings(OG.IPFSDownloadSettings, jobID)
		if err != nil {
			Fatal(fmt.Sprintf("Error processing downoad settings for job ID (%s): %s", jobID, err), 1)
			return err
		}
		err = ipfs.DownloadJob(
			ctx,
			cm,
			j.Spec.Outputs,
			results,
			processedDownloadSettings,
		)

		if err != nil {
			Fatal(fmt.Sprintf("Error downloading results from job ID (%s): %s", jobID, err), 1)
			return err
		}

		fmt.Fprintf(os.Stderr, "Results for job '%s' have been written to...\n", jobID)
		fmt.Fprintf(os.Stdout, "%s\n", processedDownloadSettings.OutputDir)

		return nil
	},
}
