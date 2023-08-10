//go:generate stringer -type=JobSelectionDataLocality -output=input_locality_type_string.go
package semantic

import (
	"fmt"
	"strings"
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
