package wasm

import (
	_ "embed"

	"github.com/ipfs/go-cid"
	ipldcodec "github.com/ipld/go-ipld-prime/codec/dagjson"
	dslschema "github.com/ipld/go-ipld-prime/schema/dsl"
	"github.com/multiformats/go-multihash"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage"
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
	cidBuilder                         = cid.V1Builder{Codec: cid.DagJSON, MhType: multihash.SHA2_256}
)

type WasmEngineSpec struct {
	// The module that contains the WASM code to start running.
	EntryModule storage.Storage `json:"EntryModule,omitempty"`

	// The name of the function in the EntryModule to call to run the job. For
	// WASI jobs, this will always be `_start`, but jobs can choose to call
	// other WASM functions instead. The EntryPoint must be a zero-parameter
	// zero-result function.
	EntryPoint string `json:"EntryPoint,omitempty"`

	// The arguments supplied to the program (i.e. as ARGV).
	Parameters []string `json:"Parameters,omitempty"`

	// The variables available in the environment of the running program.
	EnvironmentVariables []string `json:"EnvironmentVariables,omitempty"`

	// TODO #880: Other WASM modules whose exports will be available as imports
	// to the EntryModule.
	ImportModules []storage.Storage `json:"ImportModules,omitempty"`
}

func (e *WasmEngineSpec) Cid() (cid.Cid, error) {
	spec, err := e.AsSpec()
	if err != nil {
		return cid.Undef, err
	}
	return cidBuilder.Sum(spec.SchemaData)
}

func (e *WasmEngineSpec) AsSpec() (engine.Engine, error) {
	return engine.Encode(e, defaultModelEncoder, EngineSchema)
}

func Decode(spec engine.Engine) (*WasmEngineSpec, error) {
	return engine.Decode[WasmEngineSpec](spec, defaultModelDecoder)
}
