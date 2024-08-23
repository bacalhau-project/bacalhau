package types

import (
	"slices"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

var _ ConfigProvider = (*PublishersConfig)(nil)

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

func (p PublishersConfig) Installed(kind string) bool {
	_, ok := p.Config[kind]
	return ok
}

var _ ProviderType = (*LocalPublisherConfig)(nil)

type LocalPublisherConfig struct {
	Address   string `yaml:"Address"`
	Port      int    `yaml:"Port"`
	Directory string `yaml:"Directory"`
}

func (l LocalPublisherConfig) Kind() string {
	return models.PublisherLocal
}

var _ ProviderType = (*S3PublisherConfig)(nil)

type S3PublisherConfig struct {
	PreSignedURLDisabled   bool     `yaml:"PreSignedURLDisabled"`
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration"`
}

func (s S3PublisherConfig) Kind() string {
	return models.PublisherS3
}

type IpfsPublisherConfig struct {
	// Connect is the multiaddress to connect to for IPFS.
	Connect string `yaml:"Connect"`
}

func (i IpfsPublisherConfig) Kind() string {
	return models.PublisherIPFS
}
