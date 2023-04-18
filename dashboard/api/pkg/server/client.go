package server

// client code for accessing bacalhau, literally ripped straight from the CLI

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const maxWaitTime = 900

func init() { //nolint:gochecknoinits
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}
}

var realSpec model.Spec = model.Spec{
	EngineSpec: model.EngineSpec{
		Type: model.EngineDocker,
		Spec: map[string]interface{}{
			model.DockerEngineImageKey:      "ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",
			model.DockerEngineEntrypointKey: []string{"python", "main.py", "--o", "./outputs", "--p"},
		},
	},
	Verifier: model.VerifierNoop,
	PublisherSpec: model.PublisherSpec{
		Type: model.PublisherIpfs,
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
	EngineSpec: model.EngineSpec{
		Type: model.EngineDocker,
		Spec: map[string]interface{}{
			model.DockerEngineImageKey:      "ubuntu",
			model.DockerEngineEntrypointKey: []string{"echo"},
		},
	},
	Verifier: model.VerifierNoop,
	PublisherSpec: model.PublisherSpec{
		Type: model.PublisherIpfs,
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
	host, port := system.Envs[env].APIHost, system.Envs[env].APIPort
	client := publicapi.NewRequesterAPIClient(host, port)

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
	// TODO this is really gross, we can probably do better.
	if entryPoint, ok := s.EngineSpec.Spec[model.DockerEngineEntrypointKey].([]string); ok {
		entryPoint = append(entryPoint, prompt)
		s.EngineSpec.Spec[model.DockerEngineEntrypointKey] = append(s.EngineSpec.Spec[model.DockerEngineEntrypointKey].([]string), entryPoint...)
	}
	return runGenericJob(s)
}
