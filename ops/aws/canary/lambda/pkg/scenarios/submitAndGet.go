package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func SubmitAndGet(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := getClient()

	cm := system.NewCleanupManager()
	j, err := getSampleDockerJob()
	if err != nil {
		return err
	}
	submittedJob, err := client.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}

	results, err := getClientV2().Jobs().Results(&apimodels.ListJobResultsRequest{
		JobID: submittedJob.Metadata.ID,
	})
	if err != nil {
		return err
	}

	if len(results.Results) == 0 {
		return fmt.Errorf("no results found")
	}

	outputDir, err := os.MkdirTemp(os.TempDir(), "submitAndGet")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outputDir)

	downloadSettings, err := getIPFSDownloadSettings()
	if err != nil {
		return err
	}
	downloadSettings.OutputDir = outputDir

	downloaderProvider, err := util.NewStandardDownloaders(cm)
	if err != nil {
		return err
	}

	err = downloader.DownloadResults(ctx, results.Results, downloaderProvider, downloadSettings)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, downloader.DownloadFilenameStdout))
	if err != nil {
		return err
	}

	err = compareOutput(body, defaultEchoMessage)
	if err != nil {
		return err
	}

	return nil
}
