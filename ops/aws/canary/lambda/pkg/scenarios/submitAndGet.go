package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

func SubmitAndGet(ctx context.Context) error {
	// intentionally delay creation of the client so a new client is created for each
	// scenario to mimic the behavior of bacalhau cli.
	client := getClient()

	cm := system.NewCleanupManager()
	j := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, j, nil)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.Metadata.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}

	results, err := client.GetResults(ctx, submittedJob.Metadata.ID)
	if err != nil {
		return err
	}

	if len(results) == 0 {
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

	err = ipfs.DownloadJob(ctx, cm, submittedJob.Spec.Outputs, results, *downloadSettings)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, ipfs.DownloadVolumesFolderName, ipfs.DownloadFilenameStdout))
	if err != nil {
		return err
	}

	err = compareOutput(body, defaultEchoMessage)
	if err != nil {
		return err
	}

	return nil
}
