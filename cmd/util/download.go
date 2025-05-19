package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

func DownloadResultsHandler(
	ctx context.Context,
	cmd *cobra.Command,
	cfg types.Bacalhau,
	apiV2 clientv2.API,
	jobIDOrName string,
	namespace string,
	downloadSettings *cliflags.DownloaderSettings,
) error {
	cmd.PrintErrf("Fetching results of job '%s'...\n", jobIDOrName)

	request := &apimodels.ListJobResultsRequest{
		JobID: jobIDOrName,
	}
	request.Namespace = namespace
	response, err := apiV2.Jobs().Results(ctx, request)
	if err != nil {
		return errors.New(err.Error())
	}

	if len(response.Items) == 0 {
		// No results doesn't mean error, so we should print out a message and return nil
		cmd.Println("No results found")
		cmd.Println("You can check the logged output of the job using the logs command.")
		cmd.Printf("\n  bacalhau job logs %s\n", jobIDOrName)
		return nil
	}
	downloaderProvider, err := util.NewStandardDownloaders(ctx, cfg.ResultDownloaders)
	if err != nil {
		return err
	}

	// check if we don't support downloading the results
	for _, result := range response.Items {
		if !downloaderProvider.Has(ctx, result.Type) {
			cmd.PrintErrln(
				"No supported downloader found for the published results. You will have to download the results differently.")
			b, err := json.MarshalIndent(response.Items, "", "    ")
			if err != nil {
				return err
			}
			cmd.PrintErrln(string(b))
			return nil
		}
	}

	processedDownloadSettings, err := processDownloadSettings(
		downloadSettings,
		jobIDOrName,
		namespace,
	)
	if err != nil {
		return err
	}

	err = downloader.DownloadResults(
		ctx,
		response.Items,
		downloaderProvider,
		(*downloader.DownloaderSettings)(processedDownloadSettings),
	)

	if err != nil {
		return err
	}

	cmd.Printf("Results for job '%s' have been written to...\n", jobIDOrName)
	cmd.Printf("%s\n", processedDownloadSettings.OutputDir)

	return nil
}
func processDownloadSettings(
	settings *cliflags.DownloaderSettings,
	jobIDOrName string,
	namespace string,
) (*cliflags.DownloaderSettings, error) {
	if settings.OutputDir == "" {
		dir, err := ensureDefaultDownloadLocation(jobIDOrName, namespace)
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
func ensureDefaultDownloadLocation(jobIDOrName, namespace string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	downloadDir := filepath.Join(cwd, GetDefaultJobFolder(jobIDOrName, namespace))
	err = os.MkdirAll(downloadDir, AutoDownloadFolderPerm)
	if err != nil {
		return "", err
	}
	return downloadDir, nil
}

func GetDefaultJobFolder(jobIDOrName, namespace string) string {
	if namespace == "" {
		namespace = "default"
	}
	return fmt.Sprintf("job-%s-%s", namespace, jobIDOrName)
}
