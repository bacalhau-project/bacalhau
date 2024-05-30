package util

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
)

func DownloadResultsHandler(
	ctx context.Context,
	cmd *cobra.Command,
	cfg types.IpfsConfig,
	apiV2 clientv2.API,
	jobID string,
	downloadSettings *cliflags.DownloaderSettings,
) error {
	cmd.PrintErrf("Fetching results of job '%s'...\n", jobID)
	cm := GetCleanupManager(ctx)

	response, err := apiV2.Jobs().Results(ctx, &apimodels.ListJobResultsRequest{
		JobID: jobID,
	})
	if err != nil {
		Fatal(cmd, fmt.Errorf("could not get results for job %s: %w", jobID, err), 1)
	}

	if len(response.Results) == 0 {
		// No results doesn't mean error, so we should print out a message and return nil
		cmd.Println("No results found")
		cmd.Println("You can check the logged output of the job using the logs command.")
		cmd.Printf("\n  bacalhau logs %s\n", jobID)
		return nil
	}

	downloaderProvider := util.NewStandardDownloaders(cm, cfg)
	if err != nil {
		return err
	}

	// check if we don't support downloading the results
	for _, result := range response.Results {
		if !downloaderProvider.Has(ctx, result.Type) {
			cmd.PrintErrln(
				"No supported downloader found for the published results. You will have to download the results differently.")
			b, err := json.MarshalIndent(response.Results, "", "    ")
			if err != nil {
				return err
			}
			cmd.PrintErrln(string(b))
			return nil
		}
	}

	processedDownloadSettings, err := processDownloadSettings(downloadSettings, jobID)
	if err != nil {
		return err
	}

	err = downloader.DownloadResults(
		ctx,
		response.Results,
		downloaderProvider,
		(*downloader.DownloaderSettings)(processedDownloadSettings),
	)

	if err != nil {
		return err
	}

	cmd.Printf("Results for job '%s' have been written to...\n", jobID)
	cmd.Printf("%s\n", processedDownloadSettings.OutputDir)

	return nil
}
func processDownloadSettings(settings *cliflags.DownloaderSettings, jobID string) (*cliflags.DownloaderSettings, error) {
	if settings.OutputDir == "" {
		dir, err := ensureDefaultDownloadLocation(jobID)
		if err != nil {
			return settings, err
		}
		settings.OutputDir = dir
	}
	return settings, nil
}

const AutoDownloadFolderPerm = 0755

// if the user does not supply a value for "download results to here"
// then we default to making a folder in the current directory
func ensureDefaultDownloadLocation(jobID string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	downloadDir := filepath.Join(cwd, GetDefaultJobFolder(jobID))
	err = os.MkdirAll(downloadDir, AutoDownloadFolderPerm)
	if err != nil {
		return "", err
	}
	return downloadDir, nil
}

func GetDefaultJobFolder(jobID string) string {
	return fmt.Sprintf("job-%s", idgen.ShortUUID(jobID))
}
