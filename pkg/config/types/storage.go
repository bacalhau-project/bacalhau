package types

import (
	"slices"
	"strings"
)

var _ Provider = (*InputSourcesConfig)(nil)

type InputSourcesConfig struct {
	// Disabled specifies a list of storages that are disabled.
	Disabled []string `yaml:"Disabled,omitempty" json:"Disabled,omitempty"`
	// ReadTimeout specifies the maximum time allowed for reading from a storage.
	ReadTimeout Duration `yaml:"ReadTimeout,omitempty" json:"ReadTimeout,omitempty"`
	// ReadTimeout specifies the maximum number of attempts for reading from a storage.
	MaxRetryCount int               `yaml:"MaxRetryCount,omitempty" json:"MaxRetryCount,omitempty"`
	Types         InputSourcesTypes `yaml:"Types,omitempty" json:"Types,omitempty"`
}

type InputSourcesTypes struct {
	IPFS IPFSStorage `yaml:"IPFS,omitempty" json:"IPFS,omitempty"`
}

func (i InputSourcesConfig) IsNotDisabled(kind string) bool {
	return !slices.ContainsFunc(i.Disabled, func(s string) bool {
		return strings.EqualFold(s, kind)
	})
}

type IPFSStorage struct {
	// Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001
	Endpoint string `yaml:"Endpoint,omitempty" json:"Endpoint,omitempty"`
}
