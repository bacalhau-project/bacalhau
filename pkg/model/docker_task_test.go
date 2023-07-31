//go:build unit || !integration

package model

import (
	"testing"

	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalDocker(t *testing.T) {
	bytes, err := tests.ReadFile("tasks/docker_task.json")
	require.NoError(t, err)

	task, err := UnmarshalIPLD[Task](bytes, json.Decode, UCANTaskSchema)
	require.NoError(t, err)

	spec, err := task.ToSpec()
	require.NoError(t, err)

	dockerSpec, err := DecodeEngineSpec[DockerEngineSpec](spec.EngineSpec)
	require.NoError(t, err)

	engineType, err := spec.EngineSpec.Engine()
	require.NoError(t, err)

	require.Equal(t, EngineDocker, engineType)
	require.Equal(t, "ubuntu", dockerSpec.Image)
	require.Equal(t, []string{"date"}, dockerSpec.Entrypoint)
	require.Equal(t, "/", dockerSpec.WorkingDirectory)
	require.Equal(t, []string{"HELLO", "world"}, dockerSpec.EnvironmentVariables)
	require.Equal(t, []StorageSpec{}, spec.Inputs)
	require.Equal(t, []StorageSpec{
		{Path: "/outputs", Name: "outputs"},
	}, spec.Outputs)
}
