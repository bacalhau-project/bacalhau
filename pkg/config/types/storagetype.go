package types

import (
	"fmt"
	"strings"

	"go.uber.org/multierr"
	"gopkg.in/yaml.v3"
)

type JobStoreConfig struct {
	Type StorageType `yaml:"Type"`
	Path string      `yaml:"Path"`
}

func (cfg JobStoreConfig) Validate() error {
	var err error
	if cfg.Type <= UnknownStorage || cfg.Type > BoltDB {
		err = multierr.Append(err, fmt.Errorf("unknown execution store type: %q", cfg.Type.String()))
	}

	if cfg.Path == "" {
		err = multierr.Append(err, fmt.Errorf("execution store path is missing"))
	}

	return err
}

//go:generate stringer -type=StorageType
type StorageType int64

const (
	UnknownStorage StorageType = 0
	BoltDB         StorageType = 1
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
	for typ := UnknownStorage; typ <= BoltDB; typ++ {
		if equal(typ.String(), s) {
			return typ, nil
		}
	}

	return UnknownStorage, fmt.Errorf("StorageType: unknown type '%s' (valid types: %q)", s, []StorageType{BoltDB})
}

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}
