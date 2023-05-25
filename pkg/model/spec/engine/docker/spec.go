package docker

import (
	_ "embed"

	ipldcodec "github.com/ipld/go-ipld-prime/codec/dagjson"
	dslschema "github.com/ipld/go-ipld-prime/schema/dsl"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine"
)

//go:embed spec.ipldsch
var schema []byte

func load() *engine.Schema {
	dsl, err := dslschema.ParseBytes(schema)
	if err != nil {
		panic(err)
	}
	return (*engine.Schema)(dsl)
}

var (
	EngineSchema        *engine.Schema = load()
	defaultModelEncoder                = ipldcodec.Encode
	defaultModelDecoder                = ipldcodec.Decode
)

// TODO this will need to have a docker specific name inorder to avoid a collions with names.
// or we need a different semantic in spec.go
type DockerEngineSpec struct {
	// Image is the docker image to run. This must be pull-able by docker.
	Image string `json:"Image,omitempty"`

	// Entrypoint is an optional override for the default container entrypoint.
	Entrypoint []string `json:"Entrypoint,omitempty"`

	// EnvironmentVariables is a map of env to run the container with.
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`

	// WorkingDirectory is the working directory inside the container.
	WorkingDirectory string `json:"WorkingDirectory,omitempty"`
}

func (e *DockerEngineSpec) AsSpec() (engine.Engine, error) {
	return engine.Encode(e, defaultModelEncoder, EngineSchema)
}

func Decode(spec engine.Engine) (*DockerEngineSpec, error) {
	return engine.Decode[DockerEngineSpec](spec, defaultModelDecoder)
}
