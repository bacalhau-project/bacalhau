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

func (i DockerInputs) UnmarshalInto(with string, spec *Spec) error {
	spec.EngineSpec = NewDockerEngineSpec(with, i.Entrypoint, i.Env.ToStringSlice(), i.Workdir)
	spec.EngineDeprecated = EngineDocker

	inputData, err := parseInputs(i.Mounts)
	if err != nil {
		return err
	}
	spec.Inputs = inputData

	spec.Outputs = []StorageSpec{}
	for path := range i.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}
	return nil
}
