package server

// client code for accessing bacalhau, literally ripped straight from the CLI

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const maxWaitTime = 900

var (
	realEngine spec.Engine
	testEngine spec.Engine
	storage    spec.Storage
)

func init() { //nolint:gochecknoinits
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}

	realEngine, err = (&docker.DockerEngineSpec{
		Image:      "ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",
		Entrypoint: []string{"python", "main.py", "--o", "./outputs", "--p"},
	}).AsSpec()
	if err != nil {
		panic(err)
	}

	testEngine, err = (&docker.DockerEngineSpec{
		Image:      "ubuntu:latest",
		Entrypoint: []string{"echo"},
	}).AsSpec()
	if err != nil {
		panic(err)
	}

	storage, err = (&local.LocalStorageSpec{Source: "/"}).AsSpec("outputs", "/outputs")
	if err != nil {
		panic(err)
	}
}

var realSpec model.Spec = model.Spec{
	Engine:   realEngine,
	Verifier: model.VerifierNoop,
	PublisherSpec: model.PublisherSpec{
		Type: model.PublisherIpfs,
	},
	Resources: model.ResourceUsageConfig{
		GPU: "1",
	},
	Outputs: []spec.Storage{storage},
	Deal: model.Deal{
		Concurrency: 1,
	},
}

var testSpec model.Spec = model.Spec{
	Engine:   testEngine,
	Verifier: model.VerifierNoop,
	PublisherSpec: model.PublisherSpec{
		Type: model.PublisherIpfs,
	},
	Outputs: []spec.Storage{storage},
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
	mutEngine, err := docker.Mutate(s.Engine, docker.AppendEntrypoint(prompt))
	if err != nil {
		return "", err
	}
	s.Engine = mutEngine
	return runGenericJob(s)
}
