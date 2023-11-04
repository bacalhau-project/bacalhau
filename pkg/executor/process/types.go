package process

import (
	"errors"

	"github.com/hashicorp/go-multierror"
)

// EngineSpec contains necessary parameters to execute a wasm job.
type EngineSpec struct {
	Name      string   `json:"Name,omitempty"`
	Arguments []string `json:"Arguments,omitempty"`
}

func EngineSpecFromDict(m map[string]interface{}) (*EngineSpec, error) {
	e := &EngineSpec{}
	errs := new(multierror.Error)

	if name, ok := m["Name"]; !ok {
		errs = multierror.Append(errs, errors.New("Name was not found in parameter"))
	} else {
		e.Name = name.(string)
	}

	if args, ok := m["Arguments"]; !ok {
		errs = multierror.Append(errs, errors.New("Arguments were not found in parameters"))
	} else {

		for _, s := range args.([]interface{}) {
			e.Arguments = append(e.Arguments, s.(string))
		}
	}

	return e, errs.ErrorOrNil()
}
