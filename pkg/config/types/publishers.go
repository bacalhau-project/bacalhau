package types

import (
	"fmt"
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
	// Bucket specifies the S3 bucket name for managed publisher
	Bucket string `yaml:"Bucket,omitempty" json:"Bucket,omitempty"`
	// Key specifies an optional prefix for objects stored in the bucket
	Key string `yaml:"Key,omitempty" json:"Key,omitempty"`
	// Region specifies the region the S3 bucket is in
	Region string `yaml:"Region,omitempty" json:"Region,omitempty"`
	// Endpoint specifies an optional custom S3 endpoint
	Endpoint string `yaml:"Endpoint,omitempty" json:"Endpoint,omitempty"`
	// PreSignedURLExpiration specifies the duration before a pre-signed URL expires.
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration,omitempty" json:"PreSignedURLExpiration,omitempty"`
}

func (p *S3ManagedPublisher) Validate() error {
	var errs []string

	if p.Bucket == "" {
		errs = append(errs, "bucket cannot be empty")
	}

	if p.Region == "" {
		errs = append(errs, "region cannot be empty")
	}

	if p.PreSignedURLExpiration.AsTimeDuration() <= 0 {
		errs = append(errs, "pre-signed URL expiration must be greater than zero")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid publisher configuration: %s", strings.Join(errs, ", "))
	}

	return nil
}

// IsConfigured returns true if ANY specific configuration has been provided,
// even if incomplete. This helps distinguish between "not configured at all" and
// "incorrectly configured".
func (p *S3ManagedPublisher) IsConfigured() bool {
	return p != nil && (p.Bucket != "" || p.Region != "" || p.Key != "" || p.Endpoint != "")
}

type LocalPublisher struct {
	// Address specifies the endpoint the publisher serves on.
	Address string `yaml:"Address,omitempty" json:"Address,omitempty"`
	// Port specifies the port the publisher serves on.
	Port int `yaml:"Port,omitempty" json:"Port,omitempty"`
}
