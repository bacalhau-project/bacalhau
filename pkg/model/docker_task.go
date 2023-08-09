package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
)

type DockerInputs struct {
	Entrypoint []string
	Workdir    string
	Mounts     IPLDMap[string, Resource]
	Outputs    IPLDMap[string, datamodel.Node]
	Env        IPLDMap[string, string]
}

var _ JobType = (*DockerInputs)(nil)

func (docker DockerInputs) UnmarshalInto(with string, spec *Spec) error {
	envvars := make([]string, 0, len(docker.Env.Values))
	for key, val := range docker.Env.Values {
		envvars = append(envvars, key, val)
	}

	spec.EngineSpec = NewDockerEngineBuilder(with).
		WithEntrypoint(docker.Entrypoint...).
		WithWorkingDirectory(docker.Workdir).
		WithEnvironmentVariables(envvars...).
		Build()

	inputData, err := parseInputs(docker.Mounts)
	if err != nil {
		return err
	}
	spec.Inputs = inputData

	spec.Outputs = []StorageSpec{}
	for path := range docker.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}
	return nil
}
