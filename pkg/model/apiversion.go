package model

import (
	"fmt"
)

//go:generate stringer -type=APIVersion
type APIVersion int

const (
	apiVersionUnknown APIVersion = iota // must be first
	V1alpha1
	V1beta1
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
