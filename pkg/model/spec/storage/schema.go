package storage

import (
	"bytes"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec"
	ipldcodec "github.com/ipld/go-ipld-prime/codec/json"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	dmtschema "github.com/ipld/go-ipld-prime/schema/dmt"
	"github.com/multiformats/go-multihash"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/util"
)

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

var (
	defaultSchemaEncoder = ipldcodec.Encode
	defaultSchemaDecoder = ipldcodec.Decode
	cidBuilder           = cid.V1Builder{Codec: cid.DagCBOR, MhType: multihash.SHA2_256}
)

func Encode(name, mount string, params any, encoder codec.Encoder, modelSchema *Schema) (spec.Storage, error) {
	// construct a type system for the schema
	ts, err := util.NewValidatedTypeSystem((*dmtschema.Schema)(modelSchema))
	if err != nil {
		return spec.Storage{}, err
	}

	encodedParams, err := util.MarshalIPLD(params, encoder, ts)
	if err != nil {
		return spec.Storage{}, err
	}

	encodedSchema, err := modelSchema.Serialize()
	if err != nil {
		return spec.Storage{}, err
	}

	storageSpec := spec.Storage{
		Type: ts.GetSchemaType(params).Name(),
		// NB: slightly wasteful since calling Cid() calls serialize, and we just called it above, ohh well, its cheap enough for now.
		Schema:     modelSchema.Cid(),
		SchemaData: encodedSchema,
		Params:     encodedParams,
		Name:       name,
		Mount:      mount,
	}

	return storageSpec, nil
}

func Decode[P any](spec spec.Storage, decoder codec.Decoder) (*P, error) {
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

	// decode the spec parameters into the schema for as a Go object.
	return util.UnmarshalIPLD[P](spec.Params, decoder, ts)
}
