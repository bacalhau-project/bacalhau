package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/downloader/util"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// This test submits a job that uses the Docker executor with an IPFS input.
func SubmitDockerIPFSJobAndGet(ctx context.Context) error {
	client := getClient()

	cm := system.NewCleanupManager()
	j := getSampleDockerIPFSJob()

	expectedChecksum := "ea1efa312267e09809ae13f311970863  /inputs/data.tar.gz"
	expectedStat := "62731802"
	// Tests use the cid of the file we uploaded in scenarios_test.go
	if os.Getenv("BACALHAU_CANARY_TEST_CID") != "" {
		j.Spec.Inputs[0].CID = os.Getenv("BACALHAU_CANARY_TEST_CID")
		expectedChecksum = "c639efc1e98762233743a75e7798dd9c  /inputs/data.tar.gz"
		expectedStat = "21"
	}

	submittedJob, err := client.Submit(ctx, j)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return fmt.Errorf("waiting until completed: %s", err)
	}

	results, err := client.GetResults(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return fmt.Errorf("getting results: %s", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	outputDir, err := os.MkdirTemp(os.TempDir(), "submitAndGet")
	if err != nil {
		return fmt.Errorf("making temporary dir: %s", err)
	}
	defer os.RemoveAll(outputDir)

	downloadSettings, err := getIPFSDownloadSettings()
	if err != nil {
		return fmt.Errorf("getting download settings: %s", err)
	}
	downloadSettings.OutputDir = outputDir
	// canary is running every 5 minutes with a 5 minutes timeout. It should be safe to allow the download to take up to 4 minutes and leave
	// 1 minute for the rest of the test
	downloadSettings.Timeout = 240 * time.Second

	downloaderProvider := util.NewStandardDownloaders(cm, downloadSettings)
	if err != nil {
		return err
	}

	err = downloader.DownloadJob(ctx, submittedJob.Spec.Outputs, results, downloaderProvider, downloadSettings)
	if err != nil {
		return fmt.Errorf("downloading job: %s", err)
	}
	files, err := os.ReadDir(filepath.Join(downloadSettings.OutputDir, model.DownloadVolumesFolderName, j.Spec.Outputs[0].Name))
	if err != nil {
		return fmt.Errorf("reading results directory: %s", err)
	}

	for _, file := range files {
		log.Ctx(ctx).Debug().Msgf("downloaded files: %s", file.Name())
	}
	if len(files) != 3 {
		return fmt.Errorf("expected 3 files in output dir, got %d", len(files))
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, model.DownloadVolumesFolderName, j.Spec.Outputs[0].Name, "checksum.txt"))
	if err != nil {
		return err
	}

	// Tests use the checksum of the data we uploaded in scenarios_test.go
	err = compareOutput(body, expectedChecksum)
	if err != nil {
		return fmt.Errorf("testing md5 of input: %s", err)
	}
	body, err = os.ReadFile(filepath.Join(downloadSettings.OutputDir, model.DownloadVolumesFolderName, j.Spec.Outputs[0].Name, "stat.txt"))
	if err != nil {
		return err
	}
	// Tests use the stat of the data we uploaded in scenarios_test.go
	err = compareOutput(body, expectedStat)
	if err != nil {
		return fmt.Errorf("testing ls of input: %s", err)
	}

	return nil
}
