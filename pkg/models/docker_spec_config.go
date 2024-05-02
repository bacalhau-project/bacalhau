package models

const (
	EngineKeyImageDocker                = "Image"
	EngineKeyEntrypointDocker           = "Entrypoint"
	EngineKeyParametersDocker           = "Parameters"
	EngineKeyEnvironmentVariablesDocker = "EnvironmentVariables"
	EngineKeyWorkingDirectoryDocker     = "WorkingDirectory"
)

// DockerEngineSpec contains necessary parameters to execute a docker job.
type DockerEngineSpec struct {
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

// DockerSpecConfigBuilder is a struct that is used for constructing an EngineSpec object
// specifically for Docker engines using the Builder pattern.
// It embeds an SpecConfig object for handling the common builder methods.
type DockerSpecConfigBuilder struct {
	sb *SpecConfig
}

// NewDockerEngineBuilder function initializes a new DockerSpecConfigBuilder instance.
// It sets the engine type to model.EngineDocker.String() and image as per the input argument.
func DockerSpecBuilder(image string) *DockerSpecConfigBuilder {
	sb := NewSpecConfig(EngineDocker)
	sb.WithParam(EngineKeyImageDocker, image)
	return &DockerSpecConfigBuilder{sb: sb}
}

// WithEntrypoint is a builder method that sets the Docker engine entrypoint.
// It returns the DockerSpecConfigBuilder for further chaining of builder methods.
func (b *DockerSpecConfigBuilder) WithEntrypoint(e ...string) *DockerSpecConfigBuilder {
	b.sb.WithParam(EngineKeyEntrypointDocker, e)
	return b
}

// WithEnvironmentVariables is a builder method that sets the Docker engine's environment variables.
// It returns the DockerSpecConfigBuilder for further chaining of builder methods.
func (b *DockerSpecConfigBuilder) WithEnvironmentVariables(e ...string) *DockerSpecConfigBuilder {
	b.sb.WithParam(EngineKeyEnvironmentVariablesDocker, e)
	return b
}

// WithWorkingDirectory is a builder method that sets the Docker engine's working directory.
// It returns the DockerSpecConfigBuilder for further chaining of builder methods.
func (b *DockerSpecConfigBuilder) WithWorkingDirectory(e string) *DockerSpecConfigBuilder {
	b.sb.WithParam(EngineKeyWorkingDirectoryDocker, e)
	return b
}

// WithParameters is a builder method that sets the Docker engine's parameters.
// It returns the DockerSpecConfigBuilder for further chaining of builder methods.
func (b *DockerSpecConfigBuilder) WithParameters(e ...string) *DockerSpecConfigBuilder {
	b.sb.WithParam(EngineKeyParametersDocker, e)
	return b
}

// Build method constructs the final EngineSpec object by calling the embedded SpecConfig's Build method.
func (b *DockerSpecConfigBuilder) Build() (*SpecConfig, error) {
	b.sb.Normalize()
	if err := b.sb.Validate(); err != nil {
		return nil, err
	}
	return b.sb, nil
}
