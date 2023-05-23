package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec"
	ipldjson "github.com/ipld/go-ipld-prime/codec/json"
	"github.com/ipld/go-ipld-prime/node/bindnode"
	"github.com/ipld/go-ipld-prime/schema"
	dmtschema "github.com/ipld/go-ipld-prime/schema/dmt"
	"github.com/multiformats/go-multihash"
)

type Spec struct {
	Name       string
	Schema     cid.Cid
	SchemaData []byte // TODO remove when we can safely resolve the Schema cid(s) across the network.
	Params     []byte
}

var DefaultModelEncoder = ipldjson.Encode
var DefaultModelDecoder = ipldjson.Decode

var cidBuilder = cid.V1Builder{Codec: cid.Raw, MhType: multihash.SHA2_256}

type Schema dmtschema.Schema

func (s *Schema) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := DefaultModelEncoder(bindnode.Wrap(s, dmtschema.Prototypes.Schema.Type()).Representation(), buf); err != nil {
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

func Encode(params any, modelSchema *Schema) (Spec, error) {
	// TODO replace `any` with an interface that all Spec's must implement for de/serialization
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return Spec{}, err
	}

	// encode the schema and derive a cid from it
	encodedSchema, err := modelSchema.Serialize()
	if err != nil {
		return Spec{}, err
	}

	engineSpec := Spec{
		// NB: slightly wasteful since calling Cid() calls serialize, and we just called it about, ohh well, its cheap enough.
		Schema:     modelSchema.Cid(),
		SchemaData: encodedSchema,
		Params:     encodedParams,
	}

	return engineSpec, nil
}

/*
my disappointment is immeasurable and my day is ruined; dammit go: https://github.com/golang/go/issues/49085
func (s *Spec) Decode[T any]() (*T, error) {

}
*/

func Decode[P any](spec Spec) (*P, error) {
	// TODO replace `any` with an interface that all Spec's must implement for de/serialization

	// decode the spec schema.
	schemaBuilder := dmtschema.Prototypes.Schema.Representation().NewBuilder()
	if err := DefaultModelDecoder(schemaBuilder, bytes.NewReader(spec.SchemaData)); err != nil {
		return nil, err
	}

	// construct a type system for the schema
	ts := new(schema.TypeSystem)
	ts.Init()
	if err := dmtschema.Compile(ts, bindnode.Unwrap(schemaBuilder.Build()).(*dmtschema.Schema)); err != nil {
		return nil, err
	}

	if errs := ts.ValidateGraph(); len(errs) > 0 {
		return nil, multierror.Append(fmt.Errorf("valdating schema graph"), errs...)
	}

	// decode the spec parameters into the schema for as a Go object.
	return UnmarshalIPLD[P](spec.Params, DefaultModelDecoder, (*TypeSystem)(ts))
}

// TODO below methods are duplicated from model/ipld.go, unify these into a single package; careful of deps.

type TypeSystem schema.TypeSystem

// UnmarshalIPLD parses the given bytes as a Go object using the passed decoder.
// Returns an error if the object cannot be parsed.
func UnmarshalIPLD[T any](b []byte, decoder codec.Decoder, schema *TypeSystem) (*T, error) {
	t := new(T)
	_, err := ipld.Unmarshal(b, decoder, t, schema.GetSchemaType(t))
	return t, err
}

// GetSchemaTypeName returns the name of the corresponding IPLD type in the
// schema for the passed Go object. If the type cannot be in the schema, it
// returns an empty string. It may return a non-empty string even if the type is
// not in the schema.
func (s *TypeSystem) GetSchemaTypeName(obj interface{}) string {
	// Convention: all go types share the same name as their schema types
	return reflect.TypeOf(obj).Elem().Name()
}

// GetSchemaType returns the IPLD type from the schema for the passed Go object.
// If the type is not in the schema, it returns nil.
func (s *TypeSystem) GetSchemaType(obj interface{}) schema.Type {
	name := s.GetSchemaTypeName(obj)
	return (*schema.TypeSystem)(s).TypeByName(name)
}
