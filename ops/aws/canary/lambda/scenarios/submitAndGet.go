package scenarios

import (
	"context"
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
)

func SubmitAndGet(ctx context.Context, client *publicapi.APIClient) error {
	cm := system.NewCleanupManager()
	jobSpec, jobDeal := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, jobSpec, jobDeal, nil)
	log.Info().Msgf("submitted job: %s", submittedJob.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return err
	}

	results, err := client.GetResults(ctx, submittedJob.ID)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	downloadSettings, err := getIPFSDownloadSettings()
	if err != nil {
		return err
	}
	defer os.RemoveAll(downloadSettings.OutputDir)

	err = ipfs.DownloadJob(ctx, cm, submittedJob, results, *downloadSettings)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, "stdout"))
	if err != nil {
		return err
	}

	err = compareOutput(body, defaultEchoMessage)
	if err != nil {
		return err
	}

	return nil
}
