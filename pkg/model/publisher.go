package model

import (
	"fmt"
)

//go:generate stringer -type=PublisherType --trimprefix=Publisher
type PublisherType int

const (
	publisherUnknown PublisherType = iota // must be first
	PublisherNoop
	PublisherIpfs
	PublisherFilecoin
	PublisherEstuary
	publisherDone // must be last
)

func ParsePublisherType(str string) (PublisherType, error) {
	for typ := publisherUnknown + 1; typ < publisherDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return publisherUnknown, fmt.Errorf("verifier: unknown type '%s'", str)
}

func EnsurePublisherType(typ PublisherType, str string) (PublisherType, error) {
	if IsValidPublisherType(typ) {
		return typ, nil
	}
	return ParsePublisherType(str)
}

func IsValidPublisherType(publisherType PublisherType) bool {
	return publisherType > publisherUnknown && publisherType < publisherDone
}

func PublisherTypes() []PublisherType {
	var res []PublisherType
	for typ := publisherUnknown + 1; typ < publisherDone; typ++ {
		res = append(res, typ)
	}

	return res
}
