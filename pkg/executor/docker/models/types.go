package models

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/fatih/structs"
)

const (
	EngineKeyImageDocker                = "Image"
	EngineKeyEntrypointDocker           = "Entrypoint"
	EngineKeyParametersDocker           = "Parameters"
	EngineKeyEnvironmentVariablesDocker = "EnvironmentVariables"
	EngineKeyWorkingDirectoryDocker     = "WorkingDirectory"
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
	if validate.IsBlank(c.Image) {
		return errors.New("invalid docker engine params: image cannot be empty")
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
	eb *models.SpecConfig
}

// NewDockerEngineBuilder function initializes a new DockerEngineBuilder instance.
// It sets the engine type to model.EngineDocker.String() and image as per the input argument.
func NewDockerEngineBuilder(image string) *DockerEngineBuilder {
	eb := models.NewSpecConfig(models.EngineDocker)
	eb.WithParam(EngineKeyImageDocker, image)
	return &DockerEngineBuilder{eb: eb}
}

// WithEntrypoint is a builder method that sets the Docker engine entrypoint.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithEntrypoint(e ...string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyEntrypointDocker, e)
	return b
}

// WithEnvironmentVariables is a builder method that sets the Docker engine's environment variables.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithEnvironmentVariables(e ...string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyEnvironmentVariablesDocker, e)
	return b
}

// WithWorkingDirectory is a builder method that sets the Docker engine's working directory.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithWorkingDirectory(e string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyWorkingDirectoryDocker, e)
	return b
}

// WithParameters is a builder method that sets the Docker engine's parameters.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *DockerEngineBuilder) WithParameters(e ...string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyParametersDocker, e)
	return b
}

// Build method constructs the final SpecConfig object by calling the embedded EngineBuilder's Build method.
func (b *DockerEngineBuilder) Build() *models.SpecConfig {
	return b.eb
}
