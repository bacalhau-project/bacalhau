//go:build unit || !integration

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerEngineBuilder(t *testing.T) {
	image := "my_image"
	entrypoint := []string{"sh", "-c"}
	envVars := []string{"VAR1=value1", "VAR2=value2"}
	workingDir := "/app"
	params := []string{"1", "2", "3"}

	spec := NewDockerEngineBuilder(image).
		WithEntrypoint(entrypoint...).
		WithEnvironmentVariables(envVars...).
		WithWorkingDirectory(workingDir).
		WithParameters(params...).
		Build()

	require.Equal(t, EngineDocker.String(), spec.Type, "Engine type should be 'docker'")

	// Checking Parameters
	assert.Equal(t, image, spec.Params[EngineKeyImageDocker], "Image should be '%s'", image)
	assert.Equal(t, entrypoint, spec.Params[EngineKeyEntrypointDocker], "Entrypoint should match")
	assert.Equal(t, envVars, spec.Params[EngineKeyEnvironmentVariablesDocker], "Environment variables should match")
	assert.Equal(t, workingDir, spec.Params[EngineKeyWorkingDirectoryDocker], "Working directory should be '%s'", workingDir)
	assert.Equal(t, params, spec.Params[EngineKeyParametersDocker], "Parameters should be '%s'", params)

	dockerEngine, err := DecodeEngineSpec[DockerEngineSpec](spec)
	require.NoError(t, err)

	assert.Equal(t, image, dockerEngine.Image, "Image should be equal to '%s', got '%s'", image, dockerEngine.Image)
	assert.Equal(t, entrypoint, dockerEngine.Entrypoint, "Entrypoint should be equal to '%v', got '%v'", entrypoint, dockerEngine.Entrypoint)
	assert.Equal(t, envVars, dockerEngine.EnvironmentVariables, "EnvironmentVariables should be equal to '%v', got '%v'", envVars, dockerEngine.EnvironmentVariables)
	assert.Equal(t, workingDir, dockerEngine.WorkingDirectory, "WorkingDirectory should be equal to '%s', got '%s'", workingDir, dockerEngine.WorkingDirectory)
	assert.Equal(t, params, dockerEngine.Parameters, "Parameters should be equal to '%s', got '%s'", params, dockerEngine.Parameters)
}
