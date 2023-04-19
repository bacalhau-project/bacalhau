package main

import (
	typegen "github.com/whyrusleeping/cbor-gen"

	docker_spec "github.com/bacalhau-project/bacalhau/pkg/executor/docker/spec"
	wasm_spec "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/spec"
)

func main() {
	if err := typegen.WriteMapEncodersToFile("pkg/executor/docker/spec/cbor_gen.go", "spec",
		docker_spec.JobSpecDocker{},
	); err != nil {
		panic(err)
	}

	if err := typegen.WriteMapEncodersToFile("pkg/executor/wasm/spec/cbor_gen.go", "spec",
		wasm_spec.JobSpecWasm{},
		wasm_spec.KV{},
	); err != nil {
		panic(err)
	}
}
