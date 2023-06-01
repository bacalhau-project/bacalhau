package types

import (
	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
)

func EngineTypes() []cid.Cid {
	return []cid.Cid{
		docker.EngineType,
		wasm.EngineType,
	}
}

func EngineTypeNames() []string {
	return []string{
		docker.EngineType.String(),
		wasm.EngineType.String(),
	}
}
