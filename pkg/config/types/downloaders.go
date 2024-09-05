package types

import (
	"slices"
	"strings"
)

var _ Provider = (*ResultDownloaders)(nil)

type ResultDownloaders struct {
	Disabled []string               `yaml:"Disabled,omitempty"`
	Timeout  Duration               `yaml:"Timeout,omitempty"`
	Types    ResultDownloadersTypes `yaml:"Types,omitempty"`
}

type ResultDownloadersTypes struct {
	IPFS IpfsDownloader `yaml:"IPFS,omitempty"`
}

func (r ResultDownloaders) IsNotDisabled(kind string) bool {
	return !slices.ContainsFunc(r.Disabled, func(s string) bool {
		return strings.ToLower(s) == strings.ToLower(kind)
	})
}

type IpfsDownloader struct {
	// Endpoint is the multiaddress to connect to for IPFS.
	Endpoint string `yaml:"Endpoint,omitempty"`
}
