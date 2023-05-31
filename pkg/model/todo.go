package model

import (
	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
)

type S3StorageSpec struct {
	Bucket         string `json:"Bucket,omitempty"`
	Key            string `json:"Key,omitempty"`
	ChecksumSHA256 string `json:"Checksum,omitempty"`
	VersionID      string `json:"VersionID,omitempty"`
	Endpoint       string `json:"Endpoint,omitempty"`
	Region         string `json:"Region,omitempty"`
}

// PublishedStorageSpec is a wrapper for a StorageSpec that has been published
// by a compute provider - it keeps info about the host job that
// lead to the given storage spec being published
type PublishedResult struct {
	NodeID string       `json:"NodeID,omitempty"`
	Data   spec.Storage `json:"Data,omitempty"`
}

type DownloadItem struct {
	Name string
	// TODO could make a real CID
	CID        string
	URL        string
	SourceType cid.Cid
	Target     string
}
