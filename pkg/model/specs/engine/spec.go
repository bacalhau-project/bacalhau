package engine

import (
	"bytes"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec"
	ipldcodec "github.com/ipld/go-ipld-prime/codec/json"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	dmtschema "github.com/ipld/go-ipld-prime/schema/dmt"
	"github.com/multiformats/go-multihash"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/util"
)

var (
	defaultSchemaEncoder = ipldcodec.Encode
	defaultSchemaDecoder = ipldcodec.Decode
	cidBuilder           = cid.V1Builder{Codec: cid.DagJSON, MhType: multihash.SHA2_256}
)

type Spec struct {
	Schema     cid.Cid
	SchemaData []byte // TODO remove when we can safely resolve the Schema cid(s) across the network.
	Params     []byte
}

type Schema dmtschema.Schema

func (s *Schema) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := defaultSchemaEncoder(bindnode.Wrap(s, dmtschema.Prototypes.Schema.Type()).Representation(), buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Schema) Cid() cid.Cid {
	sb, err := s.Serialize()
	if err != nil {
		panic(err)
	}
	c, err := cidBuilder.Sum(sb)
	if err != nil {
		panic(err)
	}
	return c
}

func Encode(params any, encoder codec.Encoder, modelSchema *Schema) (Spec, error) {
	// construct a type system for the schema
	ts, err := util.NewValidatedTypeSystem((*dmtschema.Schema)(modelSchema))
	if err != nil {
		return Spec{}, err
	}

	encodedParams, err := util.MarshalIPLD(params, encoder, ts)
	if err != nil {
		return Spec{}, err
	}

	encodedSchema, err := modelSchema.Serialize()
	if err != nil {
		return Spec{}, err
	}

	engineSpec := Spec{
		// NB: slightly wasteful since calling Cid() calls serialize, and we just called it above, ohh well, its cheap enough for now.
		Schema:     modelSchema.Cid(),
		SchemaData: encodedSchema,
		Params:     encodedParams,
	}

	return engineSpec, nil
}

func Decode[P any](spec Spec, decoder codec.Decoder) (*P, error) {
	// decode the spec schema.
	schemaBuilder := dmtschema.Prototypes.Schema.Representation().NewBuilder()
	if err := defaultSchemaDecoder(schemaBuilder, bytes.NewReader(spec.SchemaData)); err != nil {
		return nil, err
	}

	// construct a type system for the schema
	ts, err := util.NewValidatedTypeSystem(bindnode.Unwrap(schemaBuilder.Build()).(*dmtschema.Schema))
	if err != nil {
		return nil, err
	}

	// decode the spec parameters into the schema as a Go object.
	return util.UnmarshalIPLD[P](spec.Params, decoder, ts)
}
