package types

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type JobStoreConfig struct {
	Type StorageType `yaml:"Type"`
	Path string      `yaml:"Path"`
}

//go:generate stringer -type=StorageType -linecomment
type StorageType int64

const (
	InMemory StorageType = 0
	BoltDB   StorageType = 1
)

func (j *StorageType) UnmarshalText(text []byte) error {
	out, err := ParseStorageType(string(text))
	if err != nil {
		return err
	}
	*j = out
	return nil
}

func (j StorageType) MarshalYAML() (interface{}, error) {
	return j.String(), nil
}

func (j *StorageType) UnmarshalYAML(value *yaml.Node) error {
	out, err := ParseStorageType(value.Value)
	if err != nil {
		return err
	}
	*j = out
	return nil
}

func ParseStorageType(s string) (ret StorageType, err error) {
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
