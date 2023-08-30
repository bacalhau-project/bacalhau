package util

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func DownloadResultsHandler(
	ctx context.Context,
	cmd *cobra.Command,
	jobID string,
	downloadSettings *cliflags.DownloaderSettings,
) error {
	cmd.PrintErrf("Fetching results of job '%s'...\n", jobID)
	cm := GetCleanupManager(ctx)
	j, _, err := GetAPIClient(ctx).Get(ctx, jobID)
	if err != nil {
		if _, ok := err.(*bacerrors.JobNotFound); ok {
			return err
		} else {
			Fatal(cmd, fmt.Errorf("unknown error trying to get job (ID: %s): %+w", jobID, err), 1)
		}
	}

	results, err := GetAPIClient(ctx).GetResults(ctx, j.Job.Metadata.ID)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	processedDownloadSettings, err := processDownloadSettings(downloadSettings, j.Job.Metadata.ID)
	if err != nil {
		return err
	}

	downloaderProvider := util.NewStandardDownloaders(cm, (*model.DownloaderSettings)(processedDownloadSettings))
	if err != nil {
		return err
	}

	// check if we don't support downloading the results
	for _, result := range results {
		if !downloaderProvider.Has(ctx, result.Data.StorageSource.String()) {
			cmd.PrintErrln(
				"No supported downloader found for the published results. You will have to download the results differently.")
			b, err := json.MarshalIndent(results, "", "    ")
			if err != nil {
				return err
			}
			cmd.PrintErrln(string(b))
			return nil
		}
	}

	err = downloader.DownloadResults(
		ctx,
		results,
		downloaderProvider,
		(*model.DownloaderSettings)(processedDownloadSettings),
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
	return fmt.Sprintf("job-%s", system.GetShortID(jobID))
}
