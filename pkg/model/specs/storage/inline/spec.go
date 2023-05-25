package inline

import (
	_ "embed"
	"errors"
	"fmt"

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
	EncodingError                       = errors.New("encoding InlineStorageSpec to storage.Spec")
	DecodingError                       = errors.New("decoding storage.Spec to InlineStorageSpec")
)

type InlineStorageSpec struct {
	URL string
}

func (e *InlineStorageSpec) AsSpec() (storage.Spec, error) {
	spec, err := storage.Encode(e, defaultModelEncoder, Schema)
	if err != nil {
		return storage.Spec{}, errors.Join(EncodingError, err)
	}
	return spec, nil
}

func Decode(spec storage.Spec) (*InlineStorageSpec, error) {
	if spec.Schema != Schema.Cid() {
		return nil, fmt.Errorf("unexpected spec schema %s: %w", spec, DecodingError)
	}
	out, err := storage.Decode[InlineStorageSpec](spec, defaultModelDecoder)
	if err != nil {
		return nil, errors.Join(DecodingError, err)
	}
	return out, nil
}
