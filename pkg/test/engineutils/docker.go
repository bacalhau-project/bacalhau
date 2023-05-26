package engineutils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
)

func MakeDockerJob(
	t testing.TB,
	verifierType model.Verifier,
	publisherType model.Publisher,
	entrypointArray []string) *model.Job {
	j := model.NewJob()

	j.Spec = model.Spec{
		Engine: DockerMakeEngine(t,
			DockerWithImage("ubuntu:latest"),
			DockerWithEntrypoint(entrypointArray...),
		),
		Verifier: verifierType,
		PublisherSpec: model.PublisherSpec{
			Type: publisherType,
		},
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}

	return j
}

func DockerWithImage(i string) func(d *docker.DockerEngineSpec) {
	return func(d *docker.DockerEngineSpec) {
		d.Image = i
	}
}
func DockerWithEntrypoint(e ...string) func(d *docker.DockerEngineSpec) {
	return func(d *docker.DockerEngineSpec) {
		d.Entrypoint = e
	}
}
func DockerWithWorkingDirectory(w string) func(d *docker.DockerEngineSpec) {
	return func(d *docker.DockerEngineSpec) {
		d.WorkingDirectory = w
	}
}
func DockerWithEnvironmentVariables(e ...string) func(d *docker.DockerEngineSpec) {
	return func(d *docker.DockerEngineSpec) {
		d.EnvironmentVariables = e
	}
}

func DockerMakeEngine(t testing.TB, opts ...func(d *docker.DockerEngineSpec)) spec.Engine {
	d := &docker.DockerEngineSpec{}
	for _, opt := range opts {
		opt(d)
	}
	out, err := d.AsSpec()
	require.NoError(t, err)
	return out
}

func DockerDecodeEngine(t testing.TB, engine spec.Engine) *docker.DockerEngineSpec {
	out, err := docker.Decode(engine)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
