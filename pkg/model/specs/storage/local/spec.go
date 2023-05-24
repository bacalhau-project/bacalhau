package local

import (
	_ "embed"

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

type LocalStorageSpec struct {
	Path string
}

func (e *LocalStorageSpec) AsSpec() (storage.Spec, error) {
	return storage.Encode("storage", e, defaultModelEncoder, StorageSchema)
}

func Decode(spec storage.Spec) (*LocalStorageSpec, error) {
	return storage.Decode[LocalStorageSpec](spec, defaultModelDecoder)
}
