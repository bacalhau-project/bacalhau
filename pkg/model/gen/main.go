package main

import (
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func main() {
	if err := typegen.WriteMapEncodersToFile("pkg/model/cbor_gen.go", "model",
		model.StorageSpec{},
		model.S3StorageSpec{},
		model.JobSpecWasm{},
		model.JobSpecDocker{},
		model.KV{},
	); err != nil {
		panic(err)
	}
}
