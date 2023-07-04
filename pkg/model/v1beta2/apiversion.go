package v1beta2

import (
	"fmt"
)

//go:generate stringer -type=APIVersion
type APIVersion int

const (
	apiVersionUnknown APIVersion = iota // must be first
	// V1alpha1 Deprecated but left here to preserve enum ordering
	V1alpha1
	// V1beta1 is Deprecated but left here to preserve enum ordering
	V1beta1
	// V1beta2 is the current API version
	V1beta2
	apiVersionDone // must be last
)

func APIVersionLatest() APIVersion {
	return apiVersionDone - 1
}

func ParseAPIVersion(str string) (APIVersion, error) {
	for typ := apiVersionUnknown + 1; typ < apiVersionDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return apiVersionUnknown, fmt.Errorf(
		"unknown apiversion '%s'", str)
}
