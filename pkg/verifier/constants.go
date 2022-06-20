package verifier

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=Type --trimprefix=Type
type Type int

const (
	typeUnknown Type = iota // must be first
	TypeNoop
	TypeIpfs
	typeDone // must be last
)

func ParseType(str string) (Type, error) {
	for typ := typeUnknown + 1; typ < typeDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return typeUnknown, fmt.Errorf("verifier: unknown type '%s'", str)
}

func Types() []Type {
	var res []Type
	for typ := typeUnknown + 1; typ < typeDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}
