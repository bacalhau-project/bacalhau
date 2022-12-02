package scenarios

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// This test submits a job that uses the Docker executor with an IPFS input.
//
func SubmitDockerIPFSJobAndGet(ctx context.Context) error {
	client := bacalhau.GetAPIClient()

	cm := system.NewCleanupManager()
	j := getSampleDockerIPFSJob()
	submittedJob, err := client.Submit(ctx, j, nil)
	if err != nil {
		return err
	}

	log.Info().Msgf("submitted job: %s", submittedJob.ID)

	err = waitUntilCompleted(ctx, client, submittedJob)
	if err != nil {
		return fmt.Errorf("waiting until completed: %s", err)
	}

	results, err := client.GetResults(ctx, submittedJob.ID)
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

	err = ipfs.DownloadJob(ctx, cm, submittedJob.Spec.Outputs, results, *downloadSettings)
	if err != nil {
		return fmt.Errorf("downloading job: %s", err)
	}
	files, err := osReadDir(filepath.Join(downloadSettings.OutputDir, ipfs.DownloadVolumesFolderName, j.Spec.Outputs[0].Name))
	if err != nil {
		return fmt.Errorf("reading results directory: %s", err)
	}

	for _, file := range files {
		log.Debug().Msgf("downloaded files: %s", file)
	}
	if len(files) != 2 {
		return fmt.Errorf("expected 2 files in output dir, got %d", len(files))
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, ipfs.DownloadVolumesFolderName, j.Spec.Outputs[0].Name, "checksum.txt"))
	if err != nil {
		return err
	}
	err = compareOutput(body, "07024a158889ccabb23c090a79558800  /inputs/data.tar.gz")
	if err != nil {
		return fmt.Errorf("testing md5 of input: %s", err)
	}

	return nil
}
