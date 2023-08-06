//go:build unit || !integration

package jobtransform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func TestEngineMigrator(t *testing.T) {
	ctx := context.Background()
	engineMigrator := NewEngineMigrator()

	t.Run("migrate docker", func(t *testing.T) {
		image := "my_image"
		entrypoint := []string{"sh", "-c"}
		envVars := []string{"VAR1=value1", "VAR2=value2"}
		workingDir := "/app"
		params := []string{"1", "2", "3"}
		verify := func(t *testing.T, job model.Job) {
			assert.Equal(t, model.EngineDocker, job.Spec.Engine)
			assert.Equal(t, image, job.Spec.Docker.Image)
			assert.Equal(t, params, job.Spec.Docker.Parameters)
			assert.Equal(t, entrypoint, job.Spec.Docker.Entrypoint)
			assert.Equal(t, workingDir, job.Spec.Docker.WorkingDirectory)
			assert.Equal(t, envVars, job.Spec.Docker.EnvironmentVariables)

			dockerSpec, err := model.DecodeEngineSpec[model.DockerEngineSpec](job.Spec.EngineSpec)
			require.NoError(t, err)

			assert.Equal(t, model.EngineDocker.String(), job.Spec.EngineSpec.Type)
			assert.Equal(t, image, dockerSpec.Image)
			assert.Equal(t, params, dockerSpec.Parameters)
			assert.Equal(t, entrypoint, dockerSpec.Entrypoint)
			assert.Equal(t, workingDir, dockerSpec.WorkingDirectory)
			assert.Equal(t, envVars, dockerSpec.EnvironmentVariables)
		}
		t.Run("down", func(t *testing.T) {
			dockerJob := model.Job{
				Spec: model.Spec{
					EngineSpec: model.NewDockerEngineBuilder(image).
						WithEntrypoint(entrypoint...).
						WithParameters(params...).
						WithWorkingDirectory(workingDir).
						WithEnvironmentVariables(envVars...).
						Build(),
				},
			}
			modified, err := engineMigrator(ctx, &dockerJob)
			require.NoError(t, err)
			require.True(t, modified)
			verify(t, dockerJob)

		})
		t.Run("up", func(t *testing.T) {
			dockerJob := model.Job{
				Spec: model.Spec{
					Engine: model.EngineDocker,
					Docker: model.JobSpecDocker{
						Image:                image,
						Entrypoint:           entrypoint,
						Parameters:           params,
						EnvironmentVariables: envVars,
						WorkingDirectory:     workingDir,
					},
				},
			}
			modified, err := engineMigrator(ctx, &dockerJob)
			require.NoError(t, err)
			require.True(t, modified)
			verify(t, dockerJob)
		})
	})
	t.Run("migrate wasm", func(t *testing.T) {
		entrypoint := "_start"
		parameters := []string{"arg1", "arg2"}
		envVars := map[string]string{"VAR1": "value1", "VAR2": "value2"}
		entryModule := model.StorageSpec{
			StorageSource: model.StorageSourceS3,
			Name:          "w2b",
		}
		importModules := []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				Name:          "w3b",
			},
			{
				Name:          "w4b",
				StorageSource: model.StorageSourceEstuary,
			},
		}
		verify := func(t *testing.T, job model.Job) {
			assert.Equal(t, model.EngineWasm, job.Spec.Engine)
			assert.Equal(t, entrypoint, job.Spec.Wasm.EntryPoint)
			assert.Equal(t, parameters, job.Spec.Wasm.Parameters)
			assert.Equal(t, entryModule, job.Spec.Wasm.EntryModule)
			assert.Equal(t, importModules, job.Spec.Wasm.ImportModules)
			assert.Equal(t, envVars, job.Spec.Wasm.EnvironmentVariables)

			wasmSpec, err := model.DecodeEngineSpec[model.WasmEngineSpec](job.Spec.EngineSpec)
			require.NoError(t, err)

			assert.Equal(t, model.EngineWasm.String(), job.Spec.EngineSpec.Type)
			assert.Equal(t, entrypoint, wasmSpec.Entrypoint)
			assert.Equal(t, parameters, wasmSpec.Parameters)
			assert.Equal(t, entryModule, wasmSpec.EntryModule)
			assert.Equal(t, importModules, wasmSpec.ImportModules)
			assert.Equal(t, envVars, wasmSpec.EnvironmentVariables)
		}

		t.Run("down", func(t *testing.T) {
			wasmJob := model.Job{
				Spec: model.Spec{
					EngineSpec: model.NewWasmEngineBuilder(entryModule).
						WithEntrypoint(entrypoint).
						WithParameters(parameters...).
						WithImportModules(importModules...).
						WithEnvironmentVariables(envVars).
						Build(),
				},
			}
			modified, err := engineMigrator(ctx, &wasmJob)
			require.NoError(t, err)
			require.True(t, modified)
			verify(t, wasmJob)
		})
		t.Run("up", func(t *testing.T) {
			wasmJob := model.Job{
				Spec: model.Spec{
					Engine: model.EngineWasm,
					Wasm: model.JobSpecWasm{
						EntryModule:          entryModule,
						EntryPoint:           entrypoint,
						Parameters:           parameters,
						EnvironmentVariables: envVars,
						ImportModules:        importModules,
					},
				},
			}
			modified, err := engineMigrator(ctx, &wasmJob)
			require.NoError(t, err)
			require.True(t, modified)
			verify(t, wasmJob)
		})
	})
	t.Run("migrate noop", func(t *testing.T) {
		verify := func(t *testing.T, job model.Job) {
			assert.Equal(t, model.EngineNoop, job.Spec.Engine)
			assert.Equal(t, model.EngineNoop.String(), job.Spec.EngineSpec.Type)
			assert.Empty(t, job.Spec.EngineSpec.Params)
		}
		t.Run("down", func(t *testing.T) {
			noopJob := model.Job{
				Spec: model.Spec{
					EngineSpec: model.NewEngineBuilder().
						WithType(model.EngineNoop.String()).
						Build(),
				},
			}
			modified, err := engineMigrator(ctx, &noopJob)
			require.NoError(t, err)
			require.True(t, modified)
			verify(t, noopJob)
		})
		t.Run("up", func(t *testing.T) {
			noopJob := model.Job{
				Spec: model.Spec{
					Engine: model.EngineNoop,
				},
			}
			modified, err := engineMigrator(ctx, &noopJob)
			require.NoError(t, err)
			require.True(t, modified)
			verify(t, noopJob)
		})
	})
	t.Run("do not migrate job", func(t *testing.T) {
		expectedType := "TestEngineSpec"
		expectedName := "an_engine_name"
		expectedColors := []string{"red", "blue", "green"}

		spec := model.NewEngineBuilder().
			WithType(expectedType).
			WithParam("Name", expectedName).
			WithParam("Colors", expectedColors).
			Build()
		job := model.Job{
			Spec: model.Spec{
				EngineSpec: spec,
			},
		}

		modified, err := engineMigrator(ctx, &job)
		require.NoError(t, err)
		require.False(t, modified)

		assert.Equal(t, expectedType, spec.Type, "Expected Type to be '%s', got '%s'", expectedType, spec.Type)
		assert.Equal(t, expectedName, spec.Params["Name"], "Expected Name to be '%s', got '%s'", expectedName, spec.Params["Name"])
		assert.Equal(t, expectedColors, spec.Params["Colors"], "Expected Color to be '%s', got '%s'", expectedColors, spec.Params["Color"])
	})
}
