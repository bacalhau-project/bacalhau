package s3

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

type S3StorageSpec struct {
	Bucket         string
	Key            string
	ChecksumSHA256 string
	VersionID      string
	Endpoint       string
	Region         string
}

func (e *S3StorageSpec) AsSpec() (storage.Spec, error) {
	return storage.Encode("storage", e, defaultModelEncoder, StorageSchema)
}

func Decode(spec storage.Spec) (*S3StorageSpec, error) {
	return storage.Decode[S3StorageSpec](spec, defaultModelDecoder)
}
