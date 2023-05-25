package engine

import (
	"fmt"

	"github.com/ipfs/go-cid"
)

type Engine struct {
	// Type is the name of the Go structure this Spec contains. Used to improve human readability.
	Type string
	// Schema is the CID of SchemaData.
	Schema cid.Cid
	// SchemaData is the IPLD schema encoded to bytes (deterministically). It described Params.
	SchemaData []byte // TODO remove when we can safely resolve the Schema cid(s) across the network.
	// Params is the data for a specific spec, it can be decoded using the IPLD Schema.
	Params []byte
}

func (e Engine) String() string {
	return fmt.Sprintf("[%s]:%s", e.Type, e.Schema)
}
