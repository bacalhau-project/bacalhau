package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/cbor"
	ipldjson "github.com/ipld/go-ipld-prime/codec/json"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	dmtschema "github.com/ipld/go-ipld-prime/schema/dmt"
	dslschema "github.com/ipld/go-ipld-prime/schema/dsl"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestEngineSpec struct {
	Schema     cid.Cid
	SchemaData []byte
	Spec       []byte
}

var cidBuilder = cid.V1Builder{Codec: cid.Raw, MhType: multihash.SHA2_256}

func MakeEngineSpec[P any](params P, schema []byte) (*TestEngineSpec, error) {
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	schemaDMT, err := dslschema.ParseBytes(schema)
	if err != nil {
		return nil, err
	}

	// encode the schema and derive a cid from it
	buf := new(bytes.Buffer)
	err = cbor.Encode(bindnode.Wrap(schemaDMT, dmtschema.Prototypes.Schema.Type()).Representation(), buf)
	if err != nil {
		return nil, err
	}

	encodedSchema := buf.Bytes()
	schemaCid, err := cidBuilder.Sum(encodedSchema)
	if err != nil {
		return nil, err
	}

	engineSpec := &TestEngineSpec{
		Schema:     schemaCid,
		SchemaData: encodedSchema,
		Spec:       encodedParams,
	}

	return engineSpec, nil
}

func DecodeEngineSpec[P any](engineSpec *TestEngineSpec) (*P, error) {
	schemaBuilder := dmtschema.Prototypes.Schema.Representation().NewBuilder()
	if err := cbor.Decode(schemaBuilder, bytes.NewReader(engineSpec.SchemaData)); err != nil {
		return nil, err
	}

	ts := new(schema.TypeSystem)
	ts.Init()
	if err := dmtschema.Compile(ts, bindnode.Unwrap(schemaBuilder.Build()).(*dmtschema.Schema)); err != nil {
		return nil, err
	}

	if errs := ts.ValidateGraph(); len(errs) > 0 {
		return nil, multierror.Append(fmt.Errorf("valdating schema graph"), errs...)
	}

	return UnmarshalIPLD[P](engineSpec.Spec, ipldjson.Decode, (*Schema)(ts))
}

func MakeDockerJobEngineSpec(image, workingdir string, entrypoint, envvar []string) (*TestEngineSpec, error) {
	// create and encode a docker job as bytes
	jobSpecDocker := &JobSpecDocker{
		Image:                image,
		Entrypoint:           entrypoint,
		EnvironmentVariables: envvar,
		WorkingDirectory:     workingdir,
	}
	encodedJobSpecDocker, err := json.Marshal(jobSpecDocker)
	if err != nil {
		return nil, err
	}

	// encodes to CBOR cid of bafkreicwdxl7h6jrhto7v64escrnwlselmvqzntlbweyn3gt7h5wtow46u
	dockerSchema := `
type DockerEngineSpec struct {
    Image String
    Entrypoint [String]
    EnvironmentVariables [String]
    WorkingDirectory String
}
`

	schemaDMT, err := dslschema.ParseBytes([]byte(dockerSchema))
	if err != nil {
		return nil, err
	}

	// encode the schema and derive a cid from it
	schemaNode := bindnode.Wrap(schemaDMT, dmtschema.Prototypes.Schema.Type())
	buf := new(bytes.Buffer)
	err = cbor.Encode(schemaNode.Representation(), buf)
	if err != nil {
		return nil, err
	}

	encodedJobSecDockerSchema := buf.Bytes()
	schemaCid, err := cidBuilder.Sum(encodedJobSecDockerSchema)
	if err != nil {
		return nil, err
	}

	engineSpec := &TestEngineSpec{
		Schema:     schemaCid,
		SchemaData: encodedJobSecDockerSchema,
		Spec:       encodedJobSpecDocker,
	}

	return engineSpec, nil
}

func DecodeDockerEngineSpec(engine *TestEngineSpec) (*JobSpecDocker, error) {
	// assert the schema matches what we are expecting
	expectedSchemaCID, err := cid.Decode("bafkreicwdxl7h6jrhto7v64escrnwlselmvqzntlbweyn3gt7h5wtow46u")
	if err != nil {
		return nil, err
	}
	if !engine.Schema.Equals(expectedSchemaCID) {
		return nil, fmt.Errorf("unsupported schema: %s", expectedSchemaCID)
	}

	schemaBuilder := dmtschema.Prototypes.Schema.Representation().NewBuilder()
	if err := cbor.Decode(schemaBuilder, bytes.NewReader(engine.SchemaData)); err != nil {
		return nil, err
	}
	ts := new(schema.TypeSystem)
	ts.Init()

	if err := dmtschema.Compile(ts, bindnode.Unwrap(schemaBuilder.Build()).(*dmtschema.Schema)); err != nil {
		return nil, err
	}

	if errs := ts.ValidateGraph(); len(errs) > 0 {
		return nil, multierror.Append(fmt.Errorf("valdating schema graph"), errs...)
	}

	node, err := ipld.Decode(engine.Spec, ipldjson.Decode)
	if err != nil {
		return nil, err
	}
	fmt.Println(node)
	out := new(JobSpecDocker)
	if _, err := ipld.Unmarshal(engine.Spec, ipldjson.Decode, out, ts.TypeByName("DockerEngineSpec")); err != nil {
		return nil, err
	}

	return out, nil
}

func TestTheWholeThing(t *testing.T) {
	// client
	engineSpec, err := MakeEngineSpec[JobSpecDocker](JobSpecDocker{
		Image:                "ubuntu:latest",
		Entrypoint:           []string{"date"},
		EnvironmentVariables: []string{"hello", "world"},
		WorkingDirectory:     "/",
	}, []byte(`
type JobSpecDocker struct {
    Image String
    Entrypoint [String]
    EnvironmentVariables [String]
    WorkingDirectory String
}
`))
	require.NoError(t, err)

	dockerEngine, err := DecodeEngineSpec[JobSpecDocker](engineSpec)
	require.NoError(t, err)

	assert.Equal(t, "ubuntu:latest", dockerEngine.Image)
	assert.Equal(t, []string{"date"}, dockerEngine.Entrypoint)
	assert.Equal(t, "/", dockerEngine.WorkingDirectory)
	assert.Equal(t, []string{"hello", "world"}, dockerEngine.EnvironmentVariables)

}
