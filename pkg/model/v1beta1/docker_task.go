package v1beta1

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
	spec.Engine = EngineDocker
	spec.Docker = JobSpecDocker{
		Image:            with,
		Entrypoint:       docker.Entrypoint,
		WorkingDirectory: docker.Workdir,
	}

	spec.Docker.EnvironmentVariables = []string{}
	for key, val := range docker.Env.Values {
		spec.Docker.EnvironmentVariables = append(spec.Docker.EnvironmentVariables, key, val)
	}

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
