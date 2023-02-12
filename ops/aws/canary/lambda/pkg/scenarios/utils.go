package scenarios

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"k8s.io/apimachinery/pkg/selection"
)

const defaultEchoMessage = "hello λ!"
const canaryAnnotation = "canary"

func getSampleDockerJob() *model.Job {
	var j = &model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"echo",
				defaultEchoMessage,
			},
		},
		Annotations: []string{canaryAnnotation},
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}

func getSampleDockerIPFSJob() *model.Job {
	var j = &model.Job{
		APIVersion: model.APIVersionLatest().String(),
	}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"bash",
				"-c",
				"stat --format=%s /inputs/data.tar.gz > /outputs/stat.txt && md5sum /inputs/data.tar.gz > /outputs/checksum.txt && cp /inputs/data.tar.gz /outputs/data.tar.gz && sync",
			},
		},
		Inputs: []model.StorageSpec{
			// This is a 64MB file backed by Filecoin deals via web3.storage on Phil's account
			// You can download via https://w3s.link/ipfs/bafybeihxutvxg3bw7fbwohq4gvncrk3hngkisrtkp52cu7qu7tfcuvktnq
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "inputs",
				CID:           "bafybeihxutvxg3bw7fbwohq4gvncrk3hngkisrtkp52cu7qu7tfcuvktnq",
				Path:          "/inputs/data.tar.gz",
			},
		},
		Outputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "outputs",
				Path:          "/outputs",
			},
		},
		Annotations: []string{canaryAnnotation},
		NodeSelectors: []model.LabelSelectorRequirement{
			{
				Key:      "owner",
				Operator: selection.Equals,
				Values:   []string{"bacalhau"},
			},
		},
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
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
	if apiPort == "" {
		apiPort = fmt.Sprint(system.Envs[system.GetEnvironment()].APIPort)
	}
	client := publicapi.NewRequesterAPIClient(fmt.Sprintf("http://%s:%s", apiHost, apiPort))
	return client
}
