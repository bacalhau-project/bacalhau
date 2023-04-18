package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
)

// TODO these are duplicated across the docker executor package and here to avoid dep hell, need a better solution.
const (
	DockerEngineType          = EngineDocker
	DockerEngineImageKey      = "Image"
	DockerEngineEntrypointKey = "Entrypoint"
	DockerEngineWorkDirKey    = "WorkingDirectory"
	DockerEngineEnvVarKey     = "EnvironmentVariables"
)

var _ JobType = (*DockerInputs)(nil)

type DockerInputs struct {
	Entrypoint []string
	Workdir    string
	Mounts     IPLDMap[string, Resource]
	Outputs    IPLDMap[string, datamodel.Node]
	Env        IPLDMap[string, string]
}

func (docker DockerInputs) EngineSpec(with string) (EngineSpec, error) {
	spec := make(map[string]interface{})
	spec[DockerEngineImageKey] = with
	spec[DockerEngineEntrypointKey] = docker.Entrypoint
	spec[DockerEngineWorkDirKey] = docker.Workdir
	spec[DockerEngineEnvVarKey] = docker.Env

	return EngineSpec{
		Type: EngineDocker,
		Spec: spec,
	}, nil
}

func (docker DockerInputs) InputStorageSpecs(_ string) ([]StorageSpec, error) {
	return parseInputs(docker.Mounts)
}

func (docker DockerInputs) OutputStorageSpecs(_ string) ([]StorageSpec, error) {
	outputs := make([]StorageSpec, 0, len(docker.Outputs.Values))
	for path := range docker.Outputs.Values {
		outputs = append(outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}
	return outputs, nil
}
