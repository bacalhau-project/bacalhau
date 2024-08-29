package cfgtypes

import (
	"slices"
	"strings"
)

var _ Provider = (*PublishersConfig)(nil)

type PublishersConfig struct {
	Disabled []string       `yaml:"Disabled,omitempty"`
	IPFS     IPFSPublisher  `yaml:"IPFS,omitempty"`
	S3       S3Publisher    `yaml:"S3,omitempty"`
	Local    LocalPublisher `yaml:"Local,omitempty"`
}

func (p PublishersConfig) Enabled(kind string) bool {
	return !slices.ContainsFunc(p.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

var _ Configurable = (*IPFSPublisher)(nil)

type IPFSPublisher struct {
	// Endpoint specifies the endpoint Multiaddress for the IPFS publisher
	Endpoint string `yaml:"Endpoint,omitempty"`
}

func (c IPFSPublisher) Installed() bool {
	return c != IPFSPublisher{}
}

var _ Configurable = (*S3Publisher)(nil)

type S3Publisher struct {
	// PreSignedURLDisabled specifies whether pre-signed URLs are enabled for the S3 provider.
	PreSignedURLDisabled bool `yaml:"PreSignedURLDisabled,omitempty"`
	// PreSignedURLExpiration specifies the duration before a pre-signed URL expires.
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration,omitempty"`
}

func (c S3Publisher) Installed() bool {
	return c != S3Publisher{}
}

var _ Configurable = (*LocalPublisher)(nil)

type LocalPublisher struct {
	// Address is the endpoint the publisher serves on.
	Address string `yaml:"Address,omitempty"`
	// Port is the port the publisher serves on.
	Port int `yaml:"Port,omitempty"`
	// Directory is a path to location on disk where content is served from.
	Directory string `yaml:"Directory,omitempty"`
}

func (l LocalPublisher) Installed() bool {
	return l != LocalPublisher{}
}
