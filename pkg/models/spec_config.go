package models

import (
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"golang.org/x/exp/maps"
)

type SpecConfig struct {
	// Type of the config
	Type string

	// Params is a map of the config params
	Params map[string]interface{}
}

func (s *SpecConfig) Normalize() {
	if s == nil {
		return
	}
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
