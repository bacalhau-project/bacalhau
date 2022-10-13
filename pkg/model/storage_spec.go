package model

// StorageSpec represents some data on a storage engine. Storage engines are
// specific to particular execution engines, as different execution engines
// will mount data in different ways.
type StorageSpec struct {
	// StorageSource is the abstract source of the data. E.g. a storage source
	// might be a URL download, but doesn't specify how the execution engine
	// does the download or what it will do with the downloaded data.
	StorageSource StorageSourceType `json:"StorageSource,omitempty"`

	// Name of the spec's data, for reference.
	Name string `json:"Name,omitempty"`

	// The unique ID of the data, where it makes sense (for example, in an
	// IPFS storage spec this will be the data's CID).
	// NOTE: The below is capitalized to match IPFS & IPLD (even though it's out of golang fmt)
	CID string `json:"CID,omitempty"`

	// Source URL of the data
	URL string `json:"URL,omitempty"`

	// The path that the spec's data should be mounted on, where it makes
	// sense (for example, in a Docker storage spec this will be a filesystem
	// path).
	// TODO: #668 Replace with "Path" (note the caps) for yaml/json when we update the n.js file
	Path string `json:"path,omitempty"`

	// Additional properties specific to each driver
	Metadata map[string]string `json:"Metadata,omitempty"`
}
