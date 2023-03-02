package v1beta1

import (
	"fmt"
)

//go:generate stringer -type=Verifier --trimprefix=Verifier
type Verifier int

const (
	verifierUnknown Verifier = iota // must be first
	VerifierNoop
	VerifierDeterministic
	verifierDone // must be last
)

func ParseVerifier(str string) (Verifier, error) {
	for typ := verifierUnknown + 1; typ < verifierDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return verifierUnknown, fmt.Errorf("verifier: unknown type '%s'", str)
}

func IsValidVerifier(verifierType Verifier) bool {
	return verifierType > verifierUnknown && verifierType < verifierDone
}

func VerifierTypes() []Verifier {
	var res []Verifier
	for typ := verifierUnknown + 1; typ < verifierDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func VerifierNames() []string {
	var names []string
	for _, typ := range VerifierTypes() {
		names = append(names, typ.String())
	}
	return names
}

func (v Verifier) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *Verifier) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*v, err = ParseVerifier(name)
	return
}
