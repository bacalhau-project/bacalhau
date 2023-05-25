package ipfs

import (
	_ "embed"
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
	ipldcodec "github.com/ipld/go-ipld-prime/codec/dagjson"
	dslschema "github.com/ipld/go-ipld-prime/schema/dsl"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage"
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
	Schema              *storage.Schema = load()
	defaultModelEncoder                 = ipldcodec.Encode
	defaultModelDecoder                 = ipldcodec.Decode
	EncodingError                       = errors.New("encoding IPFSStorageSpec to spec.Storage")
	DecodingError                       = errors.New("decoding spec.Storage to IPFSStorageSpec")
)

type IPFSStorageSpec struct {
	CID cid.Cid
}

func (e *IPFSStorageSpec) AsSpec() (spec.Storage, error) {
	s, err := storage.Encode(e, defaultModelEncoder, Schema)
	if err != nil {
		return spec.Storage{}, errors.Join(EncodingError, err)
	}
	return s, nil
}

func Decode(spec spec.Storage) (*IPFSStorageSpec, error) {
	if spec.Schema != Schema.Cid() {
		return nil, fmt.Errorf("unexpected spec schema %s: %w", spec, DecodingError)
	}
	out, err := storage.Decode[IPFSStorageSpec](spec, defaultModelDecoder)
	if err != nil {
		return nil, errors.Join(DecodingError, err)
	}
	return out, nil
}
