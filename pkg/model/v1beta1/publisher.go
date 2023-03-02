package v1beta1

import (
	"fmt"
)

//go:generate stringer -type=Publisher --trimprefix=Publisher
type Publisher int

const (
	publisherUnknown Publisher = iota // must be first
	PublisherNoop
	PublisherIpfs
	PublisherFilecoin
	PublisherEstuary
	publisherDone // must be last
)

func ParsePublisher(str string) (Publisher, error) {
	for typ := publisherUnknown + 1; typ < publisherDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return publisherUnknown, fmt.Errorf("verifier: unknown type '%s'", str)
}

func IsValidPublisher(publisherType Publisher) bool {
	return publisherType > publisherUnknown && publisherType < publisherDone
}

func PublisherTypes() []Publisher {
	var res []Publisher
	for typ := publisherUnknown + 1; typ < publisherDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func PublisherNames() []string {
	var names []string
	for _, typ := range PublisherTypes() {
		names = append(names, typ.String())
	}
	return names
}

func (p Publisher) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Publisher) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*p, err = ParsePublisher(name)
	return
}
