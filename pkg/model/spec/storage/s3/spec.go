package s3

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
	EncodingError                       = errors.New("encoding S3StorageSpec to storage.Spec")
	DecodingError                       = errors.New("decoding storage.Spec to S3StorageSpec")
)

type S3StorageSpec struct {
	Bucket         string
	Key            string
	ChecksumSHA256 string
	VersionID      string
	Endpoint       string
	Region         string
}

func (e *S3StorageSpec) AsSpec() (storage.Storage, error) {
	spec, err := storage.Encode(e, defaultModelEncoder, Schema)
	if err != nil {
		return storage.Storage{}, errors.Join(EncodingError, err)
	}
	return spec, nil
}

func Decode(spec storage.Storage) (*S3StorageSpec, error) {
	if spec.Schema != Schema.Cid() {
		return nil, fmt.Errorf("unexpected spec schema %s: %w", spec, DecodingError)
	}
	out, err := storage.Decode[S3StorageSpec](spec, defaultModelDecoder)
	if err != nil {
		return nil, errors.Join(DecodingError, err)
	}
	return out, nil
}
