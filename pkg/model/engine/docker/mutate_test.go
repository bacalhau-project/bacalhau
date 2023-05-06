//go:build unit || !integration

package docker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/engine/docker"
)

func TestMutations(t *testing.T) {
	expectedEngine := docker.EngineSpec{
		Image:                "ubuntu:latest",
		Entrypoint:           []string{"date"},
		EnvironmentVariables: []string{"hello", "world"},
		WorkingDirectory:     "/",
	}

	spec, err := expectedEngine.AsSpec()
	require.NoError(t, err)

	t.Run("override", func(t *testing.T) {
		mutatedSpec, err := docker.Mutate(spec,
			docker.WithImage("image:mutation"),
			docker.WithEntrypoint("echo"),
			docker.WithWorkingDirectory("/home"),
			docker.WithEnvironmentVariables("goodbye", "universe"),
		)
		require.NoError(t, err)
		require.True(t, spec.Schema.Equals(mutatedSpec.Schema))
		require.Equal(t, spec.SchemaData, mutatedSpec.SchemaData)

		actualEngine, err := docker.Decode(mutatedSpec)
		require.NoError(t, err)
		assert.NotEqual(t, spec.Params, mutatedSpec.Params)
		assert.Equal(t, "image:mutation", actualEngine.Image)
		assert.Equal(t, []string{"echo"}, actualEngine.Entrypoint)
		assert.Equal(t, "/home", actualEngine.WorkingDirectory)
		assert.Equal(t, []string{"goodbye", "universe"}, actualEngine.EnvironmentVariables)
	})

	t.Run("appended", func(t *testing.T) {
		mutatedSpec, err := docker.Mutate(spec,
			docker.AppendEntrypoint("echo"),
			docker.AppendEnvironmentVariables("goodbye", "universe"),
		)
		require.NoError(t, err)
		require.True(t, spec.Schema.Equals(mutatedSpec.Schema))
		require.Equal(t, spec.SchemaData, mutatedSpec.SchemaData)

		actualEngine, err := docker.Decode(mutatedSpec)
		require.NoError(t, err)
		assert.NotEqual(t, spec.Params, mutatedSpec.Params)
		assert.Equal(t, []string{"date", "echo"}, actualEngine.Entrypoint)
		assert.Equal(t, []string{"hello", "world", "goodbye", "universe"}, actualEngine.EnvironmentVariables)

	})

}
