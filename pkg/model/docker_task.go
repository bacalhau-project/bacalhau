package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
)

type DockerInputs struct {
	Entrypoint []string
	Workdir    string
	Mounts     IPLDMap[string, Resource]
	Outputs    IPLDMap[string, datamodel.Node]
	Env        IPLDMap[string, string]
}

var _ JobType = (*DockerInputs)(nil)

func (d DockerInputs) UnmarshalInto(with string, spec *Spec) error {
	var err error

	dockerEngine := docker.DockerEngineSpec{
		Image:                with,
		Entrypoint:           d.Entrypoint,
		WorkingDirectory:     d.Workdir,
		EnvironmentVariables: make([]string, 0, len(d.Env.Values)),
	}
	for key, val := range d.Env.Values {
		dockerEngine.EnvironmentVariables = append(dockerEngine.EnvironmentVariables, key, val)
	}
	spec.Engine, err = dockerEngine.AsSpec()
	if err != nil {
		return err
	}

	spec.Inputs, err = parseInputs(d.Mounts)
	if err != nil {
		return err
	}

	spec.Outputs = make([]StorageSpec, 0, len(d.Outputs.Values))
	for path := range d.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}
	return nil
}
