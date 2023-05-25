package spec

import (
	"fmt"

	"github.com/ipfs/go-cid"
)

type Storage struct {
	// Type is the name of the Go structure this Spec contains. Used to improve human readability.
	Type string
	// Schema is the CID of SchemaData.
	Schema cid.Cid
	// SchemaData is the IPLD schema encoded to bytes (deterministically). It described Params.
	SchemaData []byte // TODO remove when we can safely resolve the Schema cid(s) across the network.
	// Params is the data for a specific spec, it can be decoded using the IPLD Schema.
	Params []byte

	// Name is the name of the specs data for reference. Example could be a wasm module name
	Name string
	// Mount is the path that the spec's data will be mounted.
	Mount string
}

func (s Storage) String() string {
	return fmt.Sprintf("[%s]:%s", s.Type, s.Schema)
}
