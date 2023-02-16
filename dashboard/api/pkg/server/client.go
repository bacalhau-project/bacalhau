package server

// client code for accessing bacalhau, literally ripped straight from the CLI

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

const maxWaitTime = 900

func init() { //nolint:gochecknoinits
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
}

var realSpec model.Spec = model.Spec{
	Engine:    model.EngineDocker,
	Verifier:  model.VerifierNoop,
	Publisher: model.PublisherIpfs,
	Docker: model.JobSpecDocker{
		Image:      "ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",
		Entrypoint: []string{"python", "main.py", "--o", "./outputs", "--p"},
	},
	Resources: model.ResourceUsageConfig{
		GPU: "1",
	},
	Outputs: []model.StorageSpec{
		{
			Name: "outputs",
			Path: "/outputs",
		},
	},
	Deal: model.Deal{
		Concurrency: 1,
	},
}

var testSpec model.Spec = model.Spec{
	Engine:    model.EngineDocker,
	Verifier:  model.VerifierNoop,
	Publisher: model.PublisherIpfs,
	Docker: model.JobSpecDocker{
		Image:      "ubuntu",
		Entrypoint: []string{"echo"},
	},
	Outputs: []model.StorageSpec{
		{
			Name: "outputs",
			Path: "/outputs",
		},
	},
	Deal: model.Deal{
		Concurrency: 1,
	},
}

func runGenericJob(s model.Spec) (string, error) {
	j, err := model.NewJobWithSaneProductionDefaults()
	if err != nil {
		return err.Error(), err
	}
	j.Spec = s

	env := system.EnvironmentProd
	baseURI := fmt.Sprintf("http://%s:%d", system.Envs[env].APIHost, system.Envs[env].APIPort)
	client := publicapi.NewRequesterAPIClient(baseURI)

	submittedJob, err := client.Submit(context.Background(), j)
	if err != nil {
		return err.Error(), err
	}

	resolver := client.GetJobStateResolver()
	resolver.SetWaitTime(maxWaitTime, time.Second)

	err = resolver.Wait(context.Background(), submittedJob.Metadata.ID, job.WaitForSuccessfulCompletion())
	if err != nil {
		return err.Error(), err
	}

	results, err := client.GetResults(context.Background(), submittedJob.Metadata.ID)
	if err != nil {
		return err.Error(), err
	}

	for _, result := range results {
		if result.Data.CID != "" {
			return result.Data.CID, nil
		}
	}

	return "", fmt.Errorf("no results found?")
}

func runStableDiffusion(prompt string, testing bool) (string, error) {
	var s model.Spec
	if testing {
		s = testSpec
	} else {
		s = realSpec
	}
	s.Docker.Entrypoint = append(s.Docker.Entrypoint, prompt)
	return runGenericJob(s)
}
