package types

import (
	"slices"
	"strings"
)

var _ Provider = (*InputSourcesConfig)(nil)

type InputSourcesConfig struct {
	Disabled []string    `yaml:"Disabled,omitempty"`
	IPFS     IPFSStorage `yaml:"IPFS,omitempty"`
	S3       S3Storage   `yaml:"S3,omitempty"`
}

func (i InputSourcesConfig) Enabled(kind string) bool {
	return !slices.ContainsFunc(i.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

var _ Configurable = (*IPFSStorage)(nil)

type IPFSStorage struct {
	// Endpoint specifies the endpoint Multiaddress for the IPFS input source.
	Endpoint string `yaml:"Endpoint,omitempty"`
}

func (c IPFSStorage) Installed() bool {
	return c != IPFSStorage{}
}

var _ Configurable = (*S3Storage)(nil)

type S3Storage struct {
	// Endpoint specifies the endpoint URL for the S3 input source.
	Endpoint string `yaml:"Endpoint,omitempty"`
	// AccessKey specifies the access key for the S3 input source.
	AccessKey string `yaml:"AccessKey,omitempty"`
	// SecretKey specifies the secret key for the S3 input source.
	SecretKey string `yaml:"SecretKey,omitempty"`
}

func (c S3Storage) Installed() bool {
	return c != S3Storage{}
}
