package local

import (
	_ "embed"
	"errors"
	"fmt"

	ipldcodec "github.com/ipld/go-ipld-prime/codec/dagjson"
	dslschema "github.com/ipld/go-ipld-prime/schema/dsl"

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
	EncodingError                       = errors.New("encoding LocalStorageSpec to spec.Storage")
	DecodingError                       = errors.New("decoding spec.Storage to LocalStorageSpec")
)

type LocalStorageSpec struct {
	Source string
}

func (e *LocalStorageSpec) AsSpec() (storage.Storage, error) {
	spec, err := storage.Encode(e, defaultModelEncoder, Schema)
	if err != nil {
		return storage.Storage{}, errors.Join(EncodingError, err)
	}
	return spec, nil
}

func Decode(spec storage.Storage) (*LocalStorageSpec, error) {
	if spec.Schema != Schema.Cid() {
		return nil, fmt.Errorf("unexpected spec schema %s: %w", spec, DecodingError)
	}
	out, err := storage.Decode[LocalStorageSpec](spec, defaultModelDecoder)
	if err != nil {
		return nil, errors.Join(DecodingError, err)
	}
	return out, nil
}
