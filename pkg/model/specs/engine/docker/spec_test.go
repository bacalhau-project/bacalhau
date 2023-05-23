//go:build unit || !integration

package docker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/engine/docker"
)

func TestRoundTrip(t *testing.T) {
	expectedEngine := docker.EngineSpec{
		Image:                "ubuntu:latest",
		Entrypoint:           []string{"date"},
		EnvironmentVariables: []string{"hello", "world"},
		WorkingDirectory:     "/",
	}

	spec, err := expectedEngine.AsSpec()
	require.NoError(t, err)

	// TODO better assertions, probably easiest to vectorize them as constants.
	require.NotEmpty(t, spec.SchemaData)
	require.NotEmpty(t, spec.Params)
	// TODO hard code this to a real cid when we settle on the schema layout.
	require.True(t, docker.EngineSchema.Cid().Equals(spec.Schema))

	t.Log(string(spec.SchemaData))

	actualEngine, err := docker.Decode(spec)
	require.NoError(t, err)

	assert.Equal(t, expectedEngine.Image, actualEngine.Image)
	assert.Equal(t, expectedEngine.Entrypoint, actualEngine.Entrypoint)
	assert.Equal(t, expectedEngine.WorkingDirectory, actualEngine.WorkingDirectory)
	assert.Equal(t, expectedEngine.EnvironmentVariables, actualEngine.EnvironmentVariables)
}
