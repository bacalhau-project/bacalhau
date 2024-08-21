package types

import (
	"slices"
	"strings"
)

type InputSourcesConfig struct {
	Disabled []string                          `yaml:"Disabled,omitempty"`
	Config   map[string]map[string]interface{} `yaml:"Config,omitempty"`
}

func (i InputSourcesConfig) ConfigMap() map[string]map[string]interface{} {
	return i.Config
}

func (i InputSourcesConfig) Enabled(kind string) bool {
	return !slices.ContainsFunc(i.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

func (i InputSourcesConfig) HasConfig(kind string) bool {
	_, ok := i.Config[kind]
	return ok
}

var _ ProviderType = (*S3InputSourceConfig)(nil)

type S3InputSourceConfig struct {
	PreSignedURLDisabled   bool     `yaml:"PreSignedURLDisabled"`
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration"`
}

const KindStorageS3 = "s3storage"

func (s S3InputSourceConfig) Kind() string {
	return KindStorageS3
}

type IpfsInputSourceConfig struct {
	// Connect is the multiaddress to connect to for IPFS.
	Connect string `yaml:"Connect"`
}

const KindStorageIPFS = "ipfsstorage"

func (i IpfsInputSourceConfig) Kind() string {
	return KindStorageIPFS
}
