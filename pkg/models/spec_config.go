package models

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type SpecConfig struct {
	// Type of the config
	Type string `json:"Type"`

	// Params is a map of the config params
	Params map[string]interface{} `json:"Params,omitempty"`
}

// DecodeSpecConfig is a generic function that accepts an SpecConfig object.
// It marshals the SpecConfig Params into JSON and then unmarshal the JSON into a new object of type T.
// The function returns a pointer to the new object and an error object.
// If there is any issue during the JSON marshaling or unmarshalling, the function will return an error.
// TODO the double json marshaling here is inefficient, we can implement explicit per field decoding if required.
func DecodeSpecConfig[T any](spec *SpecConfig) (*T, error) {
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

func (s *SpecConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Str("type", s.Type)
	for k, v := range s.Params {
		e.Interface(k, v)
	}
}

func (s *SpecConfig) MetricAttributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("type", s.Type),
	}
}

// NewSpecConfig returns a new spec config
func NewSpecConfig(t string) *SpecConfig {
	return &SpecConfig{
		Type:   t,
		Params: make(map[string]interface{}),
	}
}

// WithParam adds a param to the spec config
func (s *SpecConfig) WithParam(key string, value interface{}) *SpecConfig {
	if s.Params == nil {
		s.Params = make(map[string]interface{})
	}
	s.Params[key] = value
	return s
}

func (s *SpecConfig) Normalize() {
	if s == nil {
		return
	}

	s.Type = strings.TrimSpace(s.Type)

	// Ensure that an empty and nil map are treated the same
	if len(s.Params) == 0 {
		s.Params = make(map[string]interface{})
	}
}

// Copy returns a shallow copy of the spec config
// TODO: implement deep copy if the value is a nested map, slice or Copyable
func (s *SpecConfig) Copy() *SpecConfig {
	if s == nil {
		return nil
	}
	return &SpecConfig{
		Type:   s.Type,
		Params: maps.Clone(s.Params),
	}
}

func (s *SpecConfig) Validate() error {
	if s == nil {
		return errors.New("nil spec config")
	}
	if validate.IsBlank(s.Type) {
		return errors.New("missing spec type")
	}
	return nil
}

// ValidateAllowBlank is the same as Validate but allows blank types.
// This is useful for when you want to validate a spec config that is optional.
func (s *SpecConfig) ValidateAllowBlank() error {
	if s == nil {
		return errors.New("nil spec config")
	}
	return nil
}

// IsType returns true if the current SpecConfig
func (s *SpecConfig) IsType(t string) bool {
	if s == nil {
		return false
	}
	t = strings.TrimSpace(t)
	return strings.EqualFold(s.Type, t)
}

// IsEmpty returns true if the spec config is empty
func (s *SpecConfig) IsEmpty() bool {
	return s == nil || (validate.IsBlank(s.Type) && len(s.Params) == 0)
}
