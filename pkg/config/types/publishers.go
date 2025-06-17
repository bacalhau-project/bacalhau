package types

import (
	"slices"
	"strings"
)

var _ Provider = (*PublishersConfig)(nil)

type PublishersConfig struct {
	// Disabled specifies a list of publishers that are disabled.
	Disabled []string       `yaml:"Disabled,omitempty" json:"Disabled,omitempty"`
	Types    PublisherTypes `yaml:"Types,omitempty" json:"Types,omitempty"`
}

type PublisherTypes struct {
	IPFS      IPFSPublisher      `yaml:"IPFS,omitempty" json:"IPFS,omitempty"`
	S3        S3Publisher        `yaml:"S3,omitempty" json:"S3,omitempty"`
	S3Managed S3ManagedPublisher `yaml:"S3Managed,omitempty" json:"S3Managed,omitempty"`
	Local     LocalPublisher     `yaml:"Local,omitempty" json:"Local,omitempty"`
}

func (p PublishersConfig) IsNotDisabled(kind string) bool {
	return !slices.ContainsFunc(p.Disabled, func(s string) bool {
		return strings.EqualFold(s, kind)
	})
}

type IPFSPublisher struct {
	// Endpoint specifies the multi-address to connect to for IPFS. e.g /ip4/127.0.0.1/tcp/5001
	Endpoint string `yaml:"Endpoint,omitempty" json:"Endpoint,omitempty"`
}

type S3Publisher struct {
	// PreSignedURLDisabled specifies whether pre-signed URLs are enabled for the S3 provider.
	PreSignedURLDisabled bool `yaml:"PreSignedURLDisabled,omitempty" json:"PreSignedURLDisabled,omitempty"`
	// PreSignedURLExpiration specifies the duration before a pre-signed URL expires.
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration,omitempty" json:"PreSignedURLExpiration,omitempty"`
}

type S3ManagedPublisher struct {
	// BucketName specifies the S3 bucket name for managed storage
	BucketName string `yaml:"BucketName,omitempty" json:"BucketName,omitempty"`
	// KeyPrefix specifies an optional prefix for keys stored in the bucket
	KeyPrefix string `yaml:"KeyPrefix,omitempty" json:"KeyPrefix,omitempty"`
	// Region specifies the region the S3 bucket is in
	Region string `yaml:"Region,omitempty" json:"Region,omitempty"`
	// Endpoint specifies an optional custom S3 endpoint
	Endpoint string `yaml:"Endpoint,omitempty" json:"Endpoint,omitempty"`
	// PreSignedURLExpiration specifies the duration before a pre-signed URL expires.
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration,omitempty" json:"PreSignedURLExpiration,omitempty"`
}

type LocalPublisher struct {
	// Address specifies the endpoint the publisher serves on.
	Address string `yaml:"Address,omitempty" json:"Address,omitempty"`
	// Port specifies the port the publisher serves on.
	Port int `yaml:"Port,omitempty" json:"Port,omitempty"`
}
