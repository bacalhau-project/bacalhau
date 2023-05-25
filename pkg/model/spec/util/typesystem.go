package util

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec"
	ipldschema "github.com/ipld/go-ipld-prime/schema"
	dmtschema "github.com/ipld/go-ipld-prime/schema/dmt"
)

// TODO below methods are duplicated from model/ipld.go, unify these into a single package; careful of deps.

type TypeSystem ipldschema.TypeSystem

// UnmarshalIPLD parses the given bytes as a Go object using the passed decoder.
// Returns an error if the object cannot be parsed.
func UnmarshalIPLD[T any](b []byte, decoder codec.Decoder, ts *TypeSystem) (*T, error) {
	t := new(T)
	_, err := ipld.Unmarshal(b, decoder, t, ts.GetSchemaType(t))
	return t, err
}

func MarshalIPLD(params any, encoder codec.Encoder, ts *TypeSystem) ([]byte, error) {
	schemaType := ts.GetSchemaType(params)
	return ipld.Marshal(encoder, params, schemaType)
}

func NewValidatedTypeSystem(schema *dmtschema.Schema) (*TypeSystem, error) {
	ts := new(ipldschema.TypeSystem)
	ts.Init()
	if err := dmtschema.Compile(ts, schema); err != nil {
		return nil, err
	}
	if errs := ts.ValidateGraph(); len(errs) > 0 {
		return nil, multierror.Append(fmt.Errorf("valdating schema graph"), errs...)
	}
	return (*TypeSystem)(ts), nil
}

// GetSchemaTypeName returns the name of the corresponding IPLD type in the
// ipldschema for the passed Go object. If the type cannot be in the ipldschema, it
// returns an empty string. It may return a non-empty string even if the type is
// not in the ipldschema.
func (s *TypeSystem) GetSchemaTypeName(obj interface{}) string {
	// Convention: all go types share the same name as their ipldschema types
	return reflect.TypeOf(obj).Elem().Name()
}

// GetSchemaType returns the IPLD type from the ipldschema for the passed Go object.
// If the type is not in the ipldschema, it returns nil.
func (s *TypeSystem) GetSchemaType(obj interface{}) ipldschema.Type {
	name := s.GetSchemaTypeName(obj)
	return (*ipldschema.TypeSystem)(s).TypeByName(name)
}
