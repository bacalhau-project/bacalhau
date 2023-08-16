package types

import (
	"fmt"
	"strings"
)

type StorageConfig struct {
	Type StorageType
	Path string
}

//go:generate stringer -type=StorageType -linecomment
type StorageType int64

const (
	InMemory StorageType = 0
	BoltDB   StorageType = 1
)

func (j *StorageType) UnmarshalText(text []byte) error {
	out, err := ParseJobStoreType(string(text))
	if err != nil {
		return err
	}
	*j = out
	return nil
}

func ParseJobStoreType(s string) (ret StorageType, err error) {
	for typ := InMemory; typ <= BoltDB; typ++ {
		if equal(typ.String(), s) {
			return typ, nil
		}
	}

	return InMemory, fmt.Errorf("StorageType: unknown type '%s'", s)
}

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}
