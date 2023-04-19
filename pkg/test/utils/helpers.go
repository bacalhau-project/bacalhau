package testutils

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/executor/docker/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func MustAsJobSpecDocker(t testing.TB, e model.EngineSpec) *spec.JobSpecDocker {
	engine, err := spec.AsJobSpecDocker(e)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}
