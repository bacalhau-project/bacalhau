package v1beta1

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
	require.Equal(t, EngineDocker, spec.Engine)
	require.Equal(t, "ubuntu", spec.Docker.Image)
	require.Equal(t, []string{"date"}, spec.Docker.Entrypoint)
	require.Equal(t, "/", spec.Docker.WorkingDirectory)
	require.Equal(t, []string{"HELLO", "world"}, spec.Docker.EnvironmentVariables)
	require.Equal(t, []StorageSpec{}, spec.Inputs)
	require.Equal(t, []StorageSpec{
		{Path: "/outputs", Name: "outputs"},
	}, spec.Outputs)
}
