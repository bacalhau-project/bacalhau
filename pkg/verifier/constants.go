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
	VerifierDeterministic
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

func EnsureVerifierType(typ VerifierType, str string) (VerifierType, error) {
	if IsValidVerifierType(typ) {
		return typ, nil
	}
	return ParseVerifierType(str)
}

func IsValidVerifierType(verifierType VerifierType) bool {
	return verifierType > verifierUnknown && verifierType < verifierDone
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
