package server

// client code for accessing bacalhau, literally ripped straight from the CLI

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

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

func runStableDiffusion(prompt string, testing bool) (string, error) {
	env := system.Production
	baseURI := fmt.Sprintf("http://%s:%d", system.Envs[env].APIHost, system.Envs[env].APIPort)
	client := publicapi.NewRequesterAPIClient(baseURI)

	j, err := model.NewJobWithSaneProductionDefaults()
	if err != nil {
		return err.Error(), err
	}
	if testing {
		j.Spec = testSpec
	} else {
		j.Spec = realSpec
	}
	j.Spec.Docker.Entrypoint = append(j.Spec.Docker.Entrypoint, prompt)

	submittedJob, err := client.Submit(context.Background(), j)
	if err != nil {
		return err.Error(), err
	}

	err = client.GetJobStateResolver().WaitUntilComplete(context.Background(), submittedJob.Metadata.ID)
	if err != nil {
		return err.Error(), err
	}

	results, err := client.GetResults(context.Background(), submittedJob.Metadata.ID)
	if err != nil {
		return err.Error(), err
	}

	for _, result := range results {
		if result.Data.Name == "outputs" {
			return result.Data.CID, nil
		}
	}

	return "", fmt.Errorf("no results found?")
}
