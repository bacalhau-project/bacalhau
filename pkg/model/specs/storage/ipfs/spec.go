package ipfs

import (
	_ "embed"

	"github.com/ipfs/go-cid"
	ipldcodec "github.com/ipld/go-ipld-prime/codec/dagjson"
	dslschema "github.com/ipld/go-ipld-prime/schema/dsl"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/storage"
)

//go:embed spec.ipldsch
var schema []byte

func load() *storage.Schema {
	dsl, err := dslschema.ParseBytes(schema)
	if err != nil {
		panic(err)
	}
	return (*storage.Schema)(dsl)
}

var (
	StorageSchema       *storage.Schema = load()
	defaultModelEncoder                 = ipldcodec.Encode
	defaultModelDecoder                 = ipldcodec.Decode
)

type IPFSStorageSpec struct {
	CID cid.Cid
}

func (e *IPFSStorageSpec) AsSpec() (storage.Spec, error) {
	return storage.Encode("storage", e, defaultModelEncoder, StorageSchema)
}

func Decode(spec storage.Spec) (*IPFSStorageSpec, error) {
	return storage.Decode[IPFSStorageSpec](spec, defaultModelDecoder)
}
