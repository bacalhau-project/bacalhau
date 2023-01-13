package scenarios

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
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
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}
	return j
}

func getIPFSDownloadSettings() (*ipfs.IPFSDownloadSettings, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	var downloadSettings *ipfs.IPFSDownloadSettings
	switch system.GetEnvironment() {
	case system.EnvironmentProd:
		downloadSettings = &ipfs.IPFSDownloadSettings{
			TimeoutSecs:    300,
			OutputDir:      dir,
			IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
		}
	case system.EnvironmentTest:
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			downloadSettings = &ipfs.IPFSDownloadSettings{
				TimeoutSecs:    300,
				OutputDir:      dir,
				IPFSSwarmAddrs: os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES"),
			}
		}
	case system.EnvironmentDev:
		log.Warn().Msg("Development environment has no download settings attached")
	case system.EnvironmentStaging:
		log.Warn().Msg("Staging environment has no download settings attached")
	}

	return downloadSettings, nil
}

func waitUntilCompleted(ctx context.Context, client *publicapi.APIClient, submittedJob *model.Job) error {
	resolver := client.GetJobStateResolver()
	totalShards := job.GetJobTotalExecutionCount(submittedJob)
	return resolver.Wait(
		ctx,
		submittedJob.Metadata.ID,
		totalShards,
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: totalShards,
		}),
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

func osReadDir(root string) ([]string, error) {
	var files []string
	f, err := os.Open(root)
	if err != nil {
		return files, err
	}
	fileInfo, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}
	return files, nil
}

func getClient() *publicapi.APIClient {
	apiHost := config.GetAPIHost()
	apiPort := config.GetAPIPort()
	if apiHost == "" {
		apiHost = system.Envs[system.Production].APIHost
	}
	if apiPort == "" {
		apiPort = fmt.Sprint(system.Envs[system.Production].APIPort)
	}
	client := publicapi.NewAPIClient(fmt.Sprintf("http://%s:%s", apiHost, apiPort))
	return client
}
