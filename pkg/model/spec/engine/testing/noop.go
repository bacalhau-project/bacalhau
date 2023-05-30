package enginetesting

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/noop"
)

func NoopMakeEngine(t testing.TB, str string) spec.Engine {
	out, err := (&noop.NoopEngineSpec{Noop: str}).AsSpec()
	require.NoError(t, err)
	return out
}
