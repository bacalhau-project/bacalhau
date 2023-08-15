package scenarios

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	pubapi "github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const defaultEchoMessage = "hello λ!"
const canaryAnnotation = "canary"

func getSampleDockerJob() (*model.Job, error) {
	nodeSelectors, err := getNodeSelectors()
	if err != nil {
		return nil, err
	}
	spec, err := job.MakeSpec(
		job.WithPublisher(model.PublisherSpec{
			Type: model.PublisherIpfs,
		}),
		job.WithEngineSpec(
			model.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint("echo", defaultEchoMessage).
				Build(),
		),
		job.WithAnnotations(canaryAnnotation),
		job.WithNodeSelector(nodeSelectors),
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
	spec, err := job.MakeSpec(
		job.WithPublisher(model.PublisherSpec{
			Type: model.PublisherIpfs,
		}),
		job.WithEngineSpec(
			model.NewDockerEngineBuilder("ubuntu").
				WithEntrypoint(
					"bash",
					"-c",
					"stat --format=%s /inputs/data.tar.gz > /outputs/stat.txt && md5sum /inputs/data.tar.gz > /outputs/checksum.txt && cp /inputs/data.tar.gz /outputs/data.tar.gz && sync",
				).Build(),
		),
		job.WithInputs(
			// This is a 64MB file backed by Filecoin deals via web3.storage on Phil's account
			// You can download via https://w3s.link/ipfs/bafybeihxutvxg3bw7fbwohq4gvncrk3hngkisrtkp52cu7qu7tfcuvktnq
			model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "inputs",
				CID:           "bafybeihxutvxg3bw7fbwohq4gvncrk3hngkisrtkp52cu7qu7tfcuvktnq",
				Path:          "/inputs/data.tar.gz",
			},
		),
		job.WithOutputs(
			model.StorageSpec{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		),
		job.WithAnnotations(canaryAnnotation),
		job.WithNodeSelector(nodeSelectors),
	)
	if err != nil {
		return nil, err
	}
	j.Spec = spec
	return j, nil
}

func getIPFSDownloadSettings() (*model.DownloaderSettings, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	IPFSSwarmAddrs := os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES")
	if IPFSSwarmAddrs == "" {
		IPFSSwarmAddrs = strings.Join(system.Envs[system.GetEnvironment()].IPFSSwarmAddresses, ",")
	}

	return &model.DownloaderSettings{
		Timeout:        50 * time.Second,
		OutputDir:      dir,
		IPFSSwarmAddrs: IPFSSwarmAddrs,
	}, nil
}

func waitUntilCompleted(ctx context.Context, client *publicapi.RequesterAPIClient, submittedJob *model.Job) error {
	resolver := client.GetJobStateResolver()
	return resolver.Wait(
		ctx,
		submittedJob.Metadata.ID,
		job.WaitForSuccessfulCompletion(),
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

func getClient() *publicapi.RequesterAPIClient {
	apiHost := config.GetAPIHost()
	apiPort := config.GetAPIPort()
	if apiHost == "" {
		apiHost = system.Envs[system.GetEnvironment()].APIHost
	}
	if apiPort == nil {
		defaultPort := system.Envs[system.GetEnvironment()].APIPort
		apiPort = &defaultPort
	}

	// Allow canaries to make an insecure connection
	return publicapi.NewRequesterAPIClient(apiHost, *apiPort, &pubapi.ClientTLSConfig{})
}

func getNodeSelectors() ([]model.LabelSelectorRequirement, error) {
	nodeSelectors := os.Getenv("BACALHAU_NODE_SELECTORS")
	if nodeSelectors != "" {
		return parse.NodeSelector(nodeSelectors)
	}
	return nil, nil
}
