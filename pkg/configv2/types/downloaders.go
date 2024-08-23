package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ResultDownloaders struct {
	Timeout Duration                          `yaml:"Timeout,omitempty"`
	Config  map[string]map[string]interface{} `yaml:"Config,omitempty"`
}

func (r ResultDownloaders) Enabled(kind string) bool {
	// TODO(review): do we want to allow downloaders to be disabled?
	// for now they are all enabled by default
	return true
}

func (r ResultDownloaders) Installed(kind string) bool {
	_, ok := r.Config[kind]
	return ok
}

func (r ResultDownloaders) ConfigMap() map[string]map[string]interface{} {
	return r.Config
}

type IpfsDownloadConfig struct {
	// Connect is the multiaddress to connect to for IPFS.
	Connect string `yaml:"Connect"`
}

func (i IpfsDownloadConfig) Kind() string {
	return models.StorageSourceIPFS
}
