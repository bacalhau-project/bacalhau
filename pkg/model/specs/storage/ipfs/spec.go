package ipfs

import (
	_ "embed"
	"errors"
	"fmt"

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
	Schema              *storage.Schema = load()
	defaultModelEncoder                 = ipldcodec.Encode
	defaultModelDecoder                 = ipldcodec.Decode
	EncodingError                       = errors.New("encoding IPFSStorageSpec to storage.Spec")
	DecodingError                       = errors.New("decoding storage.Spec to IPFSStorageSpec")
)

type IPFSStorageSpec struct {
	CID cid.Cid
}

func (e *IPFSStorageSpec) AsSpec() (storage.Spec, error) {
	spec, err := storage.Encode(e, defaultModelEncoder, Schema)
	if err != nil {
		return storage.Spec{}, errors.Join(EncodingError, err)
	}
	return spec, nil
}

func Decode(spec storage.Spec) (*IPFSStorageSpec, error) {
	if spec.Schema != Schema.Cid() {
		return nil, fmt.Errorf("unexpected spec schema %s: %w", spec, DecodingError)
	}
	out, err := storage.Decode[IPFSStorageSpec](spec, defaultModelDecoder)
	if err != nil {
		return nil, errors.Join(DecodingError, err)
	}
	return out, nil
}
