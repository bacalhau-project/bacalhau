package model

import (
	"fmt"
)

type Publisher int

const (
	publisherUnknown Publisher = iota // must be first
	PublisherNoop
	PublisherIpfs
	PublisherEstuary
	PublisherS3
	publisherDone // must be last
)

var publisherNames = map[Publisher]string{
	PublisherNoop:    "noop",
	PublisherIpfs:    "ipfs",
	PublisherEstuary: "estuary",
	PublisherS3:      "s3",
}

func ParsePublisher(str string) (Publisher, error) {
	for typ := publisherUnknown + 1; typ < publisherDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return publisherUnknown, fmt.Errorf("publisher: unknown type '%s'", str)
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

func (p Publisher) String() string {
	value, ok := publisherNames[p]
	if !ok {
		return Unknown
	}
	return value
}

func (p Publisher) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Publisher) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*p, err = ParsePublisher(name)
	return
}
