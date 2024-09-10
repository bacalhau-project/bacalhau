package types

import (
	"slices"
	"strings"
)

var _ Provider = (*PublishersConfig)(nil)

type PublishersConfig struct {
	// Disabled specifies a list of publishers that are disabled.
	Disabled []string       `yaml:"Disabled,omitempty"`
	Types    PublisherTypes `yaml:"Types,omitempty"`
}

type PublisherTypes struct {
	IPFS  IPFSPublisher  `yaml:"IPFS,omitempty"`
	S3    S3Publisher    `yaml:"S3,omitempty"`
	Local LocalPublisher `yaml:"Local,omitempty"`
}

func (p PublishersConfig) IsNotDisabled(kind string) bool {
	return !slices.ContainsFunc(p.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

type IPFSPublisher struct {
	// Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001
	Endpoint string `yaml:"Endpoint,omitempty"`
}

type S3Publisher struct {
	// PreSignedURLDisabled specifies whether pre-signed URLs are enabled for the S3 provider.
	PreSignedURLDisabled bool `yaml:"PreSignedURLDisabled,omitempty"`
	// PreSignedURLExpiration specifies the duration before a pre-signed URL expires.
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration,omitempty"`
}

type LocalPublisher struct {
	// Address specifies the endpoint the publisher serves on.
	Address string `yaml:"Address,omitempty"`
	// Port specifies the port the publisher serves on.
	Port int `yaml:"Port,omitempty"`
	// Directory specifies a path to location on disk where content is served from.
	Directory string `yaml:"Directory,omitempty"`
}
