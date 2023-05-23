//go:build unit || !integration

package docker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	docker2 "github.com/bacalhau-project/bacalhau/pkg/model/specs/engine/docker"
)

func TestMutations(t *testing.T) {
	expectedEngine := docker2.EngineSpec{
		Image:                "ubuntu:latest",
		Entrypoint:           []string{"date"},
		EnvironmentVariables: []string{"hello", "world"},
		WorkingDirectory:     "/",
	}

	spec, err := expectedEngine.AsSpec()
	require.NoError(t, err)

	t.Run("override", func(t *testing.T) {
		mutatedSpec, err := docker2.Mutate(spec,
			docker2.WithImage("image:mutation"),
			docker2.WithEntrypoint("echo"),
			docker2.WithWorkingDirectory("/home"),
			docker2.WithEnvironmentVariables("goodbye", "universe"),
		)
		require.NoError(t, err)
		require.True(t, spec.Schema.Equals(mutatedSpec.Schema))
		require.Equal(t, spec.SchemaData, mutatedSpec.SchemaData)

		actualEngine, err := docker2.Decode(mutatedSpec)
		require.NoError(t, err)
		assert.NotEqual(t, spec.Params, mutatedSpec.Params)
		assert.Equal(t, "image:mutation", actualEngine.Image)
		assert.Equal(t, []string{"echo"}, actualEngine.Entrypoint)
		assert.Equal(t, "/home", actualEngine.WorkingDirectory)
		assert.Equal(t, []string{"goodbye", "universe"}, actualEngine.EnvironmentVariables)
	})

	t.Run("appended", func(t *testing.T) {
		mutatedSpec, err := docker2.Mutate(spec,
			docker2.AppendEntrypoint("echo"),
			docker2.AppendEnvironmentVariables("goodbye", "universe"),
		)
		require.NoError(t, err)
		require.True(t, spec.Schema.Equals(mutatedSpec.Schema))
		require.Equal(t, spec.SchemaData, mutatedSpec.SchemaData)

		actualEngine, err := docker2.Decode(mutatedSpec)
		require.NoError(t, err)
		assert.NotEqual(t, spec.Params, mutatedSpec.Params)
		assert.Equal(t, []string{"date", "echo"}, actualEngine.Entrypoint)
		assert.Equal(t, []string{"hello", "world", "goodbye", "universe"}, actualEngine.EnvironmentVariables)

	})

}
