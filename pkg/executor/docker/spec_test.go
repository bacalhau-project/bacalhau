//go:build unit || !integration

package docker_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func TestLocalRoundTrip(t *testing.T) {
	image := "image"
	entrypoint := []string{"entry", "point"}
	envvars := []string{"FOO", "BAR"}
	workdir := "work"
	t.Run("happy path", func(t *testing.T) {
		engineSpec := docker.NewEngineSpec(image, entrypoint, envvars, workdir)

		dockerEngine, err := docker.AsEngine(engineSpec)
		require.NoError(t, err)

		assert.Equal(t, image, dockerEngine.Image)
		assert.Equal(t, entrypoint, dockerEngine.Entrypoint)
		assert.Equal(t, envvars, dockerEngine.EnvironmentVariables)
		assert.Equal(t, workdir, dockerEngine.WorkingDirectory)
	})
	t.Run("missing image", func(t *testing.T) {
		engineSpec := docker.NewEngineSpec("", entrypoint, envvars, workdir)

		dockerEngine, err := docker.AsEngine(engineSpec)
		require.NoError(t, err)

		assert.Equal(t, "", dockerEngine.Image)
		assert.Equal(t, entrypoint, dockerEngine.Entrypoint)
		assert.Equal(t, envvars, dockerEngine.EnvironmentVariables)
		assert.Equal(t, workdir, dockerEngine.WorkingDirectory)
	})
	t.Run("missing working directory", func(t *testing.T) {
		engineSpec := docker.NewEngineSpec(image, entrypoint, envvars, "")

		dockerEngine, err := docker.AsEngine(engineSpec)
		require.NoError(t, err)

		assert.Equal(t, image, dockerEngine.Image)
		assert.Equal(t, entrypoint, dockerEngine.Entrypoint)
		assert.Equal(t, envvars, dockerEngine.EnvironmentVariables)
		assert.Empty(t, dockerEngine.WorkingDirectory)
	})
	t.Run("missing entrypoint", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			engineSpec := docker.NewEngineSpec(image, nil, envvars, workdir)

			dockerEngine, err := docker.AsEngine(engineSpec)
			require.NoError(t, err)

			assert.Equal(t, image, dockerEngine.Image)
			assert.Empty(t, dockerEngine.Entrypoint)
			assert.Equal(t, envvars, dockerEngine.EnvironmentVariables)
			assert.Equal(t, workdir, dockerEngine.WorkingDirectory)
		})
		t.Run("empty slice", func(t *testing.T) {
			engineSpec := docker.NewEngineSpec(image, []string{}, envvars, workdir)

			dockerEngine, err := docker.AsEngine(engineSpec)
			require.NoError(t, err)

			assert.Equal(t, image, dockerEngine.Image)
			assert.Empty(t, dockerEngine.Entrypoint)
			assert.Equal(t, envvars, dockerEngine.EnvironmentVariables)
			assert.Equal(t, workdir, dockerEngine.WorkingDirectory)
		})
	})
	t.Run("missing environment variables", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			engineSpec := docker.NewEngineSpec(image, entrypoint, nil, workdir)

			dockerEngine, err := docker.AsEngine(engineSpec)
			require.NoError(t, err)

			assert.Equal(t, image, dockerEngine.Image)
			assert.Equal(t, entrypoint, dockerEngine.Entrypoint)
			assert.Empty(t, dockerEngine.EnvironmentVariables)
			assert.Equal(t, workdir, dockerEngine.WorkingDirectory)
		})
		t.Run("empty slice", func(t *testing.T) {
			engineSpec := docker.NewEngineSpec(image, entrypoint, []string{}, workdir)

			dockerEngine, err := docker.AsEngine(engineSpec)
			require.NoError(t, err)

			assert.Equal(t, image, dockerEngine.Image)
			assert.Equal(t, entrypoint, dockerEngine.Entrypoint)
			assert.Empty(t, dockerEngine.EnvironmentVariables)
			assert.Equal(t, workdir, dockerEngine.WorkingDirectory)
		})
	})
}

func TestRemoteRoundTrip(t *testing.T) {
	image := "image"
	entrypoint := []string{"entry", "point"}
	envvars := []string{"FOO", "BAR"}
	workdir := "work"
	t.Run("happy path", func(t *testing.T) {
		clientEngineSpec := docker.NewEngineSpec(image, entrypoint, envvars, workdir)

		// simulate an API call from client to server.
		engineBytes, err := json.Marshal(clientEngineSpec)
		require.NoError(t, err)

		var serverEngineSpec model.EngineSpec
		err = json.Unmarshal(engineBytes, &serverEngineSpec)
		require.NoError(t, err)

		dockerEngine, err := docker.AsEngine(serverEngineSpec)
		require.NoError(t, err)

		assert.Equal(t, image, dockerEngine.Image)
		assert.Equal(t, entrypoint, dockerEngine.Entrypoint)
		assert.Equal(t, envvars, dockerEngine.EnvironmentVariables)
		assert.Equal(t, workdir, dockerEngine.WorkingDirectory)
	})
}
