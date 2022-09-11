package scenarios

import (
	"context"
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"io/ioutil"
	"strings"
)

const defaultEchoMessage = "hello λ!"

func getSampleDockerJob() (model.JobSpec, model.JobDeal) {
	jobSpec := model.JobSpec{
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
	}

	jobDeal := model.JobDeal{
		Concurrency: 1,
	}
	return jobSpec, jobDeal
}

func getIPFSDownloadSettings() (*ipfs.IPFSDownloadSettings, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return &ipfs.IPFSDownloadSettings{
		TimeoutSecs:    60,
		OutputDir:      dir,
		IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
	}, nil
}

func waitUntilCompleted(ctx context.Context, client *publicapi.APIClient, submittedJob model.Job) error {
	resolver := client.GetJobStateResolver()
	totalShards := job.GetJobTotalExecutionCount(submittedJob)
	return resolver.Wait(
		ctx,
		submittedJob.ID,
		totalShards,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateCancelled,
			model.JobStateError,
		}),
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
