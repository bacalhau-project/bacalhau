//go:build unit || !integration

package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestDockerEngineBuilder_RoundTrip(t *testing.T) {
	testCases := []struct {
		name         string
		builder      func() *DockerEngineBuilder
		expectedSpec EngineSpec
	}{
		{
			name: "valid spec all fields",
			builder: func() *DockerEngineBuilder {
				return NewDockerEngineBuilder("myimage").
					WithEntrypoint("bash", "-c").
					WithEnvironmentVariables("KEY1=VALUE1", "KEY2=VALUE2").
					WithWorkingDirectory("/app").
					WithParameters("arg1", "arg2")
			},
			expectedSpec: EngineSpec{
				Image:                "myimage",
				Entrypoint:           []string{"bash", "-c"},
				EnvironmentVariables: []string{"KEY1=VALUE1", "KEY2=VALUE2"},
				WorkingDirectory:     "/app",
				Parameters:           []string{"arg1", "arg2"},
			},
		},
		{
			name: "valid spec no entry point",
			builder: func() *DockerEngineBuilder {
				return NewDockerEngineBuilder("myimage").
					WithEnvironmentVariables("KEY1=VALUE1", "KEY2=VALUE2").
					WithWorkingDirectory("/app").
					WithParameters("arg1", "arg2")
			},
			expectedSpec: EngineSpec{
				Image:                "myimage",
				EnvironmentVariables: []string{"KEY1=VALUE1", "KEY2=VALUE2"},
				WorkingDirectory:     "/app",
				Parameters:           []string{"arg1", "arg2"},
			},
		},
		{
			name: "valid spec no env var",
			builder: func() *DockerEngineBuilder {
				return NewDockerEngineBuilder("myimage").
					WithEntrypoint("bash", "-c").
					WithWorkingDirectory("/app").
					WithParameters("arg1", "arg2")
			},
			expectedSpec: EngineSpec{
				Image:            "myimage",
				Entrypoint:       []string{"bash", "-c"},
				WorkingDirectory: "/app",
				Parameters:       []string{"arg1", "arg2"},
			},
		},
		{
			name: "valid spec no params",
			builder: func() *DockerEngineBuilder {
				return NewDockerEngineBuilder("myimage").
					WithEntrypoint("bash", "-c").
					WithEnvironmentVariables("KEY1=VALUE1", "KEY2=VALUE2").
					WithWorkingDirectory("/app")
			},
			expectedSpec: EngineSpec{
				Image:                "myimage",
				Entrypoint:           []string{"bash", "-c"},
				EnvironmentVariables: []string{"KEY1=VALUE1", "KEY2=VALUE2"},
				WorkingDirectory:     "/app",
			},
		},
		{
			name: "valid spec no working dir",
			builder: func() *DockerEngineBuilder {
				return NewDockerEngineBuilder("myimage").
					WithEntrypoint("bash", "-c").
					WithEnvironmentVariables("KEY1=VALUE1", "KEY2=VALUE2").
					WithParameters("arg1", "arg2")
			},
			expectedSpec: EngineSpec{
				Image:                "myimage",
				Entrypoint:           []string{"bash", "-c"},
				EnvironmentVariables: []string{"KEY1=VALUE1", "KEY2=VALUE2"},
				Parameters:           []string{"arg1", "arg2"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := tc.builder()
			spec, err := builder.Build()
			require.NoError(t, err)
			assert.Equal(t, models.EngineDocker, spec.Type)

			engineSpec, err := DecodeSpec(spec)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedSpec, engineSpec)

			specBytes, err := json.Marshal(spec)
			require.NoError(t, err)

			unmarshalled := new(models.SpecConfig)
			err = json.Unmarshal(specBytes, unmarshalled)
			require.NoError(t, err)

			rtEngineSpec, err := DecodeSpec(unmarshalled)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedSpec, rtEngineSpec)

		})
	}
}

func TestEngineSpec_Validate(t *testing.T) {
	tests := []struct {
		name             string
		engineSpec       EngineSpec
		expectedErrorMsg string
	}{
		{
			name: "Valid EngineSpec",
			engineSpec: EngineSpec{
				Image:                "valid-image",
				Entrypoint:           []string{"entrypoint"},
				Parameters:           []string{"param1", "param2"},
				EnvironmentVariables: []string{"KEY1=VALUE1", "KEY2=VALUE2"},
				WorkingDirectory:     "/valid/path",
			},
			expectedErrorMsg: "",
		},
		{
			name: "Empty Image",
			engineSpec: EngineSpec{
				Image: "",
			},
			expectedErrorMsg: "invalid docker engine param: 'Image' cannot be empty",
		},
		{
			name: "Invalid Working Directory",
			engineSpec: EngineSpec{
				Image:            "valid-image",
				WorkingDirectory: "relative/path",
			},
			expectedErrorMsg: "invalid docker engine param: 'WorkingDirectory' (\"relative/path\") must contain absolute path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.engineSpec.Validate()
			if tt.expectedErrorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedErrorMsg)
			}
		})
	}
}
