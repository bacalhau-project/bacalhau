package scenarios

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
)

const defaultEchoMessage = "hello λ!"
const canaryAnnotation = "canary"

func getSampleDockerJob() (*model.Job, error) {
	nodeSelectors, err := getNodeSelectors()
	if err != nil {
		return nil, err
	}
	spec, err := legacy_job.MakeSpec(
		legacy_job.WithPublisher(model.PublisherSpec{
			Type: model.PublisherLocal,
		}),
		legacy_job.WithEngineSpec(
			model.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint("echo", defaultEchoMessage).
				Build(),
		),
		legacy_job.WithAnnotations(canaryAnnotation),
		legacy_job.WithNodeSelector(nodeSelectors),
	)
	if err != nil {
		return nil, err
	}
	var j = &model.Job{
		APIVersion: model.APIVersionLatest().String(),
		Spec:       spec,
	}
	return j, nil
}

func getDownloadSettings() (*downloader.DownloaderSettings, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "")
	if err != nil {
		return nil, err
	}

	return &downloader.DownloaderSettings{
		Timeout:   50 * time.Second,
		OutputDir: dir,
	}, nil
}

func waitUntilCompleted(ctx context.Context, client *client.APIClient, submittedJob *model.Job) error {
	resolver := client.GetJobStateResolver()
	return resolver.Wait(
		ctx,
		submittedJob.Metadata.ID,
		legacy_job.WaitForSuccessfulCompletion(),
	)
}

func compareOutput(output []byte, expectedOutput string) error {
	outputStr := string(output)
	outputStr = strings.TrimRight(outputStr, "\n")

	if outputStr != expectedOutput {
		return fmt.Errorf("output mismatch: expected '%v' but got '%v'", expectedOutput, outputStr)
	}
	return nil
}

func getNodeSelectors() ([]model.LabelSelectorRequirement, error) {
	nodeSelectors := os.Getenv("BACALHAU_NODE_SELECTORS")
	if nodeSelectors != "" {
		return parse.NodeSelector(nodeSelectors)
	}
	return nil, nil
}
