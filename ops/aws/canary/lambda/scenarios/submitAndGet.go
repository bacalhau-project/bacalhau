package scenarios

import (
	"context"
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"os"
	"path/filepath"
)

func SubmitAndGet(ctx context.Context, client *publicapi.APIClient) error {
	cm := system.NewCleanupManager()
	jobSpec, jobDeal := getSampleDockerJob()
	submittedJob, err := client.Submit(ctx, jobSpec, jobDeal, nil)
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

	downloadSettings := getIPFSDownloadSettings()
	err = ipfs.DownloadJob(ctx, cm, submittedJob, results, *downloadSettings)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, "stdout"))
	if err != nil {
		return err
	}

	if string(body) != "hello" {
		return fmt.Errorf("unexpected output: %s", body)
	}

	return nil
}
