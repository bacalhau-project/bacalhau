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

	engine, err := AsJobSpecDocker(spec.EngineSpec)
	require.NoError(t, err)

	require.Equal(t, EngineDocker, spec.EngineSpec.Type)
	require.Equal(t, "ubuntu", engine.Image)
	require.Equal(t, []string{"date"}, engine.Entrypoint)
	require.Equal(t, "/", engine.WorkingDirectory)
	require.Equal(t, []string{"HELLO", "world"}, engine.EnvironmentVariables)
	require.Equal(t, []StorageSpec{}, spec.Inputs)
	require.Equal(t, []StorageSpec{
		{Path: "/outputs", Name: "outputs"},
	}, spec.Outputs)
}
