package model

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

// DockerEngineBuilder is a struct that is used for constructing an EngineSpec object
// specifically for Docker engines using the Builder pattern.
// It embeds an EngineBuilder object for handling the common builder methods.
type DockerEngineBuilder struct {
	eb *EngineBuilder
}

// NewDockerEngineBuilder function initializes a new DockerEngineBuilder instance.
// It sets the engine type to model.EngineDocker.String() and image as per the input argument.
func NewDockerEngineBuilder(image string) *DockerEngineBuilder {
	eb := new(EngineBuilder)
	eb.WithType(EngineDocker.String())
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

// Build method constructs the final EngineSpec object by calling the embedded EngineBuilder's Build method.
func (b *DockerEngineBuilder) Build() EngineSpec {
	return b.eb.Build()
}
