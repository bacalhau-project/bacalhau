package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	cmdutil "github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func SubmitAndGet(ctx context.Context, cfg types.BacalhauConfig) error {
	// intentionally delay creation of the api client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	apiV1, err := cmdutil.GetAPIClient(cfg)
	if err != nil {
		return err
	}
	apiv2 := clientv2.New(fmt.Sprintf("http://%s:%d", cfg.Node.ClientAPI.Host, cfg.Node.ClientAPI.Port))

	cm := system.NewCleanupManager()
	j, err := getSampleDockerJob()
	if err != nil {
		return err
	}
	submittedJob, err := apiV1.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, apiV1, submittedJob)
	if err != nil {
		return err
	}

	results, err := apiv2.Jobs().Results(ctx, &apimodels.ListJobResultsRequest{
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

	downloaderProvider := util.NewStandardDownloaders(cm, cfg.Node.IPFS)
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
