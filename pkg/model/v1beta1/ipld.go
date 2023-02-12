package v1beta1

import (
	"bytes"
	"embed"
	"reflect"

	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec"
	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/schema"
)

//go:embed schemas
var schemas embed.FS

const (
	ucanTaskSchemaPath     string = "schemas/invocation.ipldsch"
	bacalhauTaskSchemaPath string = "schemas/bacalhau.ipldsch"
)

func load(path string) *Schema {
	file, err := schemas.Open(path)
	if err != nil {
		panic(err)
	}

	schema, err := ipld.LoadSchema(path, file)
	if err != nil {
		panic(err)
	}
	return (*Schema)(schema)
}

type Schema schema.TypeSystem

var (
	// The UCAN Task schema is the standardized Invocation IPLD schema, defined
	// by https://github.com/ucan-wg/invocation.
	UCANTaskSchema *Schema = load(ucanTaskSchemaPath)

	// The Bacalhau schema includes the Bacalhau specific extensions to the UCAN
	// Task IPLD spec, i.e. input structures for specific job types.
	BacalhauTaskSchema *Schema = load(bacalhauTaskSchemaPath)
)

// GetSchemaTypeName returns the name of the corresponding IPLD type in the
// schema for the passed Go object. If the type cannot be in the schema, it
// returns an empty string. It may return a non-empty string even if the type is
// not in the schema.
func (s *Schema) GetSchemaTypeName(obj interface{}) string {
	// Convention: all go types share the same name as their schema types
	return reflect.TypeOf(obj).Elem().Name()
}

// GetSchemaType returns the IPLD type from the schema for the passed Go object.
// If the type is not in the schema, it returns nil.
func (s *Schema) GetSchemaType(obj interface{}) schema.Type {
	name := s.GetSchemaTypeName(obj)
	return (*schema.TypeSystem)(s).TypeByName(name)
}

// UnmarshalIPLD parses the given bytes as a Go object using the passed decoder.
// Returns an error if the object cannot be parsed.
func UnmarshalIPLD[T any](b []byte, decoder codec.Decoder, schema *Schema) (*T, error) {
	t := new(T)
	_, err := ipld.Unmarshal(b, decoder, t, schema.GetSchemaType(t))
	return t, err
}

// Reinterpret re-parses the datamodel.Node as an object of the defined type.
func Reinterpret[T any](node datamodel.Node, schema *Schema) (*T, error) {
	// This is obviously slightly hacky and slow. but it is the most fool-proof
	// way of doing this at time of writing, because go-ipld-prime cannot handle
	// an algorithm using bindnode.Prototype and builder.AssignNode
	schemaType := schema.GetSchemaType((*T)(nil))

	var buf bytes.Buffer
	var val = new(T)
	err := json.Encode(node, &buf)
	if err != nil {
		return val, err
	}

	_, err = ipld.Unmarshal(buf.Bytes(), json.Decode, val, schemaType)
	return val, err
}

// IPLD Maps are parsed by the ipld library into structures of this type rather
// than just plain Go maps.
type IPLDMap[K comparable, V any] struct {
	Keys   []K
	Values map[K]V
}
