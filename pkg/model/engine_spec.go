package model

import (
	"encoding/json"
	"fmt"
)

type EngineSpec struct {
	Type   string
	Params map[string]interface{}
}

func (e EngineSpec) String() string {
	return e.Type
}

func (e EngineSpec) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		// Custom formatting for %v
		fmt.Fprintf(f, `Type:"%s",Params:"%+v"`, e.Type, e.Params)

	default:
		// Default to String method for %s or other formats
		fmt.Fprint(f, e.String())
	}
}

// Serialize method for the EngineSpec struct takes no arguments.
// If the Params field of the EngineSpec is nil, it returns an empty byte slice and a nil error.
// Otherwise, it attempts to convert the EngineSpec instance into a JSON-formatted byte slice.
// If successful, it returns the byte slice and a nil error.
// If an error occurs during the JSON marshaling process, it returns an empty byte slice and the error.
func (e EngineSpec) Serialize() ([]byte, error) {
	if e.Params == nil {
		return []byte{}, nil
	}
	return json.Marshal(e)
}

func (e EngineSpec) Engine() (Engine, error) {
	return ParseEngine(e.Type)
}

// DeserializeEngineSpec takes a byte slice as input, attempts to unmarshal it into an EngineSpec struct.
// If the unmarshalling is successful, it returns the populated EngineSpec and a nil error.
// In case of any error during the unmarshalling process, it returns an empty EngineSpec and the error.
func DeserializeEngineSpec(in []byte) (EngineSpec, error) {
	var out EngineSpec
	if err := json.Unmarshal(in, &out); err != nil {
		return EngineSpec{}, err
	}
	return out, nil
}

// DecodeEngineSpec is a generic function that accepts an EngineSpec object.
// It marshals the EngineSpec Params into JSON and then unmarshals the JSON into a new object of type T.
// The function returns a pointer to the new object and an error object.
// If there is any issue during the JSON marshaling or unmarshaling, the function will return an error.
// TODO the double json marshaling here is inefficient, we can implement explicit per field decoding if required.
func DecodeEngineSpec[T any](spec EngineSpec) (*T, error) {
	params, err := json.Marshal(spec.Params)
	if err != nil {
		return nil, err
	}

	out := new(T)
	if err := json.Unmarshal(params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// NewEngineBuilder function creates a new instance of the EngineBuilder.
func NewEngineBuilder() *EngineBuilder {
	return &EngineBuilder{}
}

// EngineBuilder is a struct used for constructing an EngineSpec object using the Builder pattern.
// The options field is a slice of functions that modify the EngineSpec object.
type EngineBuilder struct {
	options []func(spec *EngineSpec)
}

// add is a helper method that appends a function to the options field in the EngineBuilder.
func (b *EngineBuilder) add(cb func(spec *EngineSpec)) {
	b.options = append(b.options, cb)
}

// WithType is a builder method that sets the Type field of the EngineSpec.
// It returns the EngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithType(t string) *EngineBuilder {
	b.add(func(spec *EngineSpec) {
		spec.Type = t
	})
	return b
}

// WithParam is a builder method that sets a key-value pair in the Params field of the EngineSpec.
// It returns the EngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithParam(key string, value interface{}) *EngineBuilder {
	b.add(func(spec *EngineSpec) {
		spec.Params[key] = value
	})
	return b
}

// Build method constructs the final EngineSpec object.
// It applies all the functions stored in the options slice to the EngineSpec and returns it.
func (b *EngineBuilder) Build() EngineSpec {
	out := &EngineSpec{
		Type:   "<UNDEFINED>",
		Params: map[string]interface{}{},
	}
	for _, opt := range b.options {
		opt(out)
	}
	return *out
}
