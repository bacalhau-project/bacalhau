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

type DockerEngineBuilder struct {
	eb *EngineBuilder
}

func NewDockerEngineBuilder(image string) *DockerEngineBuilder {
	eb := new(EngineBuilder)
	eb.WithType(EngineDocker.String())
	eb.WithParam(EngineKeyImageDocker, image)
	return &DockerEngineBuilder{eb: eb}
}

func (b *DockerEngineBuilder) WithEntrypoint(e ...string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyEntrypointDocker, e)
	return b
}

func (b *DockerEngineBuilder) WithEnvironmentVariables(e ...string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyEnvironmentVariablesDocker, e)
	return b
}

func (b *DockerEngineBuilder) WithWorkingDirectory(e string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyWorkingDirectoryDocker, e)
	return b
}

func (b *DockerEngineBuilder) WithParameters(e ...string) *DockerEngineBuilder {
	b.eb.WithParam(EngineKeyParametersDocker, e)
	return b
}

func (b *DockerEngineBuilder) Build() EngineSpec {
	return b.eb.Build()
}
