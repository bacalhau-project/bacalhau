//go:generate stringer -type=JobSelectionDataLocality -output=input_locality_type_string.go
package semantic

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type JobSelectionDataLocality int64

const (
	Local    JobSelectionDataLocality = 0 // local
	Anywhere JobSelectionDataLocality = 1 // anywhere
)

func ParseJobSelectionDataLocality(s string) (ret JobSelectionDataLocality, err error) {
	for typ := Local; typ <= Anywhere; typ++ {
		if strings.EqualFold(strings.TrimSpace(typ.String()), s) {
			return typ, nil
		}
	}

	return Local, fmt.Errorf("%T: unknown type '%s'", Local, s)
}

func (i JobSelectionDataLocality) MarshalYAML() (interface{}, error) {
	return i.String(), nil
}

func (i *JobSelectionDataLocality) UnmarshalYAML(value *yaml.Node) error {
	out, err := ParseJobSelectionDataLocality(value.Value)
	if err != nil {
		return err
	}
	*i = out
	return nil
}
