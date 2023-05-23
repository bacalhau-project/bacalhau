package s3

import (
	_ "embed"

	dslschema "github.com/ipld/go-ipld-prime/schema/dsl"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/engine"
)

//go:embed spec.ipldsch
var schema []byte

func load() *engine.Schema {
	dsl, err := dslschema.ParseBytes(schema)
	if err != nil {
		panic(err)
	}
	return (*engine.Schema)(dsl)
}

var (
	StorageSchema *engine.Schema = load()
)

type S3StorageSpec struct {
	Bucket         string
	Key            string
	ChecksumSHA256 string
	VersionID      string
	Endpoint       string
	Region         string
}

func (e *S3StorageSpec) AsSpec() (engine.Spec, error) {
	return engine.Encode(e, StorageSchema)
}

func Decode(spec engine.Spec) (*S3StorageSpec, error) {
	return engine.Decode[S3StorageSpec](spec)
}
