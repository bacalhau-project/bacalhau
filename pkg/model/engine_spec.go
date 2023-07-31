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
		fmt.Fprintf(f, `Type:"%s",Params:{`, e.Type)
		keys := make([]string, 0, len(e.Params))
		for field := range e.Params {
			keys = append(keys, field)
		}

		for i, field := range keys {
			value := e.Params[field]
			if i < len(keys)-1 {
				fmt.Fprintf(f, `"%s":%v,`, field, value)
			} else {
				fmt.Fprintf(f, `"%s":%v`, field, value) // No trailing comma
			}
		}
		fmt.Fprintf(f, "}")

	default:
		// Default to String method for %s or other formats
		fmt.Fprint(f, e.String())
	}
}
func (e EngineSpec) Serialize() ([]byte, error) {
	if e.Params == nil {
		return []byte{}, nil
	}
	return json.Marshal(e)
}

func (e EngineSpec) Engine() (Engine, error) {
	return ParseEngine(e.Type)
}

func DeserializeEngineSpec(in []byte) (EngineSpec, error) {
	var out EngineSpec
	if err := json.Unmarshal(in, &out); err != nil {
		return EngineSpec{}, err
	}
	return out, nil
}

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

func NewEngineBuilder() *EngineBuilder {
	return &EngineBuilder{}
}

type EngineBuilder struct {
	options []func(spec *EngineSpec)
}

func (b *EngineBuilder) add(cb func(spec *EngineSpec)) {
	b.options = append(b.options, cb)
}

func (b *EngineBuilder) WithType(t string) *EngineBuilder {
	b.add(func(spec *EngineSpec) {
		spec.Type = t
	})
	return b
}

func (b *EngineBuilder) WithParam(key string, value interface{}) *EngineBuilder {
	b.add(func(spec *EngineSpec) {
		spec.Params[key] = value
	})
	return b
}

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
