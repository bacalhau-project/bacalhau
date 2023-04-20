package model

import (
	"fmt"
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
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
	return (&JobSpecDocker{
		Image:                with,
		Entrypoint:           docker.Entrypoint,
		EnvironmentVariables: FlattenIPLDMap(docker.Env),
		WorkingDirectory:     docker.Workdir,
	}).AsEngineSpec(), nil
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

func FlattenIPLDMap[K comparable, V any](ipldMap IPLDMap[K, V]) []string {
	flatMap := []string{}
	for _, key := range ipldMap.Keys {
		value := ipldMap.Values[key]

		// Convert key and value to string
		keyString := fmt.Sprintf("%v", key)
		valueString := fmt.Sprintf("%v", value)

		// Append to flatMap
		flatMap = append(flatMap, keyString)
		flatMap = append(flatMap, valueString)
	}

	return flatMap
}
