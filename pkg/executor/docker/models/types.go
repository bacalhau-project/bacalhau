package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/structs"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// EngineSpec contains necessary parameters to execute a docker job.
type EngineSpec struct {
	// Image this should be pullable by docker
	Image string `json:"Image,omitempty"`
	// Entrypoint optionally override the default entrypoint
	Entrypoint []string `json:"Entrypoint,omitempty"`
	// Parameters holds additional commandline arguments
	Parameters []string `json:"Parameters,omitempty"`
	// EnvironmentVariables is a slice of env to run the container with
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`
	// WorkingDirectory inside the container
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

func (c EngineSpec) Validate() error {
	if len(c.Image) == 0 {
		return fmt.Errorf("invalid docker engine param: 'Image' cannot be empty")
	}
	if c.WorkingDirectory != "" {
		if !strings.HasPrefix(c.WorkingDirectory, "/") {
			// This mirrors the implementation at path/filepath/path_unix.go#L13 which
			// we reuse here to get cross-platform working dir detection. This is
			// necessary (rather than using IsAbs()) because clients may be running on
			// Windows/Plan9 but we want to check inside Docker (linux).
			return fmt.Errorf("invalid docker engine param: 'WorkingDirectory' (%q) "+
				"must contain absolute path", c.WorkingDirectory)
		}
	}
	// Validate environment variables
	for _, env := range c.EnvironmentVariables {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) > 0 {
			if strings.HasPrefix(strings.ToUpper(parts[0]), models.EnvVarPrefix) {
				return fmt.Errorf("invalid docker engine param: environment variable '%s' cannot start with %s",
					parts[0], models.EnvVarPrefix)
			}
		}
	}
	return nil
}

func (c EngineSpec) ToMap() map[string]interface{} {
	return structs.Map(c)
}

func DecodeSpec(spec *models.SpecConfig) (EngineSpec, error) {
	if !spec.IsType(models.EngineDocker) {
		return EngineSpec{}, errors.New("invalid docker engine type. expected " + models.EngineDocker + ", but received: " + spec.Type)
	}
	inputParams := spec.Params
	if inputParams == nil {
		return EngineSpec{}, errors.New("invalid docker engine params. cannot be nil")
	}

	paramsBytes, err := json.Marshal(inputParams)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to encode docker engine specs. %w", err)
	}

	var c *EngineSpec
	err = json.Unmarshal(paramsBytes, &c)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to decode docker engine specs. %w", err)
	}
	return *c, c.Validate()
}

// DockerEngineBuilder is a struct that is used for constructing an EngineSpec object
// specifically for Docker engines using the Builder pattern.
// It embeds an EngineBuilder object for handling the common builder methods.
type DockerEngineBuilder struct {
	spec *EngineSpec
}

// NewDockerEngineBuilder function initializes a new DockerEngineBuilder instance.
// It sets the engine type to model.EngineDocker.String() and image as per the input argument.
func NewDockerEngineBuilder(image string) *DockerEngineBuilder {
	spec := &EngineSpec{
		Image: image,
	}
	return &DockerEngineBuilder{spec: spec}
}

// WithEntrypoint is a builder method that sets the Docker engine entrypoint.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithEntrypoint(e ...string) *DockerEngineBuilder {
	b.spec.Entrypoint = e
	return b
}

// WithEnvironmentVariables is a builder method that sets the Docker engine's environment variables.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithEnvironmentVariables(e ...string) *DockerEngineBuilder {
	b.spec.EnvironmentVariables = e
	return b
}

// WithWorkingDirectory is a builder method that sets the Docker engine's working directory.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithWorkingDirectory(e string) *DockerEngineBuilder {
	b.spec.WorkingDirectory = e
	return b
}

// WithParameters is a builder method that sets the Docker engine's parameters.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithParameters(e ...string) *DockerEngineBuilder {
	b.spec.Parameters = e
	return b
}

// Build method constructs the final SpecConfig object by calling the embedded EngineBuilder's Build method.
func (b *DockerEngineBuilder) Build() (*models.SpecConfig, error) {
	if err := b.spec.Validate(); err != nil {
		return nil, fmt.Errorf("building docker engine spec: %w", err)
	}
	return &models.SpecConfig{
		Type:   models.EngineDocker,
		Params: b.spec.ToMap(),
	}, nil
}

func (b *DockerEngineBuilder) MustBuild() *models.SpecConfig {
	spec, err := b.Build()
	if err != nil {
		panic(err)
	}
	return spec
}
