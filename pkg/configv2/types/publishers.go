package types

import (
	"slices"
	"strings"
)

type PublishersConfig struct {
	Disabled []string                          `yaml:"Disabled,omitempty"`
	Config   map[string]map[string]interface{} `yaml:"Config,omitempty"`
}

func (p PublishersConfig) ConfigMap() map[string]map[string]interface{} {
	return p.Config
}

func (p PublishersConfig) Enabled(kind string) bool {
	return !slices.ContainsFunc(p.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

func (p PublishersConfig) HasConfig(kind string) bool {
	_, ok := p.Config[kind]
	return ok
}

var _ ProviderType = (*LocalPublisherConfig)(nil)

type LocalPublisherConfig struct {
	Address   string `yaml:"Address"`
	Port      int    `yaml:"Port"`
	Directory string `yaml:"Directory"`
}

const KindPublisherLocal = "localpublisher"

func (l LocalPublisherConfig) Kind() string {
	return KindPublisherLocal
}

const KindPublisherS3 = "s3publisher"

type IpfsPublisherConfig struct {
	// Connect is the multiaddress to connect to for IPFS.
	Connect string `yaml:"Connect"`
}

const KindPublisherIPFS = "ipfspublisher"

func (i IpfsPublisherConfig) Kind() string {
	return KindPublisherIPFS
}
