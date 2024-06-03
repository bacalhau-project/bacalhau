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
			Type: model.PublisherIpfs,
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

func getSampleDockerIPFSJob() (*model.Job, error) {
	nodeSelectors, err := getNodeSelectors()
	if err != nil {
		return nil, err
	}
	var j = &model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}
	spec, err := legacy_job.MakeSpec(
		legacy_job.WithPublisher(model.PublisherSpec{
			Type: model.PublisherIpfs,
		}),
		legacy_job.WithEngineSpec(
			model.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint(
					"bash",
					"-c",
					"stat --format=%s /inputs/data.tar.gz > /outputs/stat.txt && md5sum /inputs/data.tar.gz > /outputs/checksum.txt && cp /inputs/data.tar.gz /outputs/data.tar.gz && sync",
				).Build(),
		),
		legacy_job.WithInputs(
			// This is a 64MB file backed by Filecoin deals via web3.storage on Phil's account
			// You can download via https://w3s.link/ipfs/bafybeihxutvxg3bw7fbwohq4gvncrk3hngkisrtkp52cu7qu7tfcuvktnq
			model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "inputs",
				CID:           "bafybeihxutvxg3bw7fbwohq4gvncrk3hngkisrtkp52cu7qu7tfcuvktnq",
				Path:          "/inputs/data.tar.gz",
			},
		),
		legacy_job.WithOutputs(
			model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		),
		legacy_job.WithAnnotations(canaryAnnotation),
		legacy_job.WithNodeSelector(nodeSelectors),
	)
	if err != nil {
		return nil, err
	}
	j.Spec = spec
	return j, nil
}

func getIPFSDownloadSettings() (*downloader.DownloaderSettings, error) {
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
