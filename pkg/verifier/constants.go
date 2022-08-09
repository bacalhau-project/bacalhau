package verifier

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=VerifierType --trimprefix=Verifier
type VerifierType int

const (
	verifierUnknown VerifierType = iota // must be first
	VerifierNoop
	verifierDone // must be last
)

func ParseVerifierType(str string) (VerifierType, error) {
	for typ := verifierUnknown + 1; typ < verifierDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return verifierUnknown, fmt.Errorf("verifier: unknown type '%s'", str)
}

func VerifierTypes() []VerifierType {
	var res []VerifierType
	for typ := verifierUnknown + 1; typ < verifierDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}
