package types

import (
	"slices"
	"strings"
)

var _ Provider = (*InputSourcesConfig)(nil)

type InputSourcesConfig struct {
	// Disabled specifies a list of storages that are disabled.
	Disabled []string `yaml:"Disabled,omitempty"`
	// ReadTimeout specifies the maximum time allowed for reading from a storage.
	ReadTimeout Duration `yaml:"ReadTimeout,omitempty"`
	// ReadTimeout specifies the maximum number of attempts for reading from a storage.
	MaxRetryCount int               `yaml:"MaxRetryCount,omitempty"`
	Types         InputSourcesTypes `yaml:"Types,omitempty"`
}

type InputSourcesTypes struct {
	IPFS IPFSStorage `yaml:"IPFS,omitempty"`
	S3   S3Storage   `yaml:"S3,omitempty"`
}

func (i InputSourcesConfig) IsNotDisabled(kind string) bool {
	return !slices.ContainsFunc(i.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

type IPFSStorage struct {
	// Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001
	Endpoint string `yaml:"Endpoint,omitempty"`
}

type S3Storage struct {
	// Endpoint specifies the endpoint URL for the S3 input source.
	Endpoint string `yaml:"Endpoint,omitempty"`
	// AccessKey specifies the access key for the S3 input source.
	AccessKey string `yaml:"AccessKey,omitempty"`
	// SecretKey specifies the secret key for the S3 input source.
	SecretKey string `yaml:"SecretKey,omitempty"`
}
