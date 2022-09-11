package scenarios

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"strings"
)

func getSampleDockerJob() (model.JobSpec, model.JobDeal) {
	jobSpec := model.JobSpec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		Docker: model.JobSpecDocker{
			Image: "ubuntu",
			Entrypoint: []string{
				"echo",
				"hello",
			},
		},
	}

	jobDeal := model.JobDeal{
		Concurrency: 1,
	}
	return jobSpec, jobDeal
}

func getIPFSDownloadSettings() *ipfs.IPFSDownloadSettings {
	return &ipfs.IPFSDownloadSettings{
		TimeoutSecs:    60,
		OutputDir:      "/tmp",
		IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
	}
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
