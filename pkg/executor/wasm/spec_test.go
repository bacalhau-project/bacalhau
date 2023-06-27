//go:build unit || !integration

package wasm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type MarshallerTestCase[T any] struct {
	Name        string
	Marshaller  func(t T) ([]byte, error)
	Unmarshaler func(b []byte, t *T) error
}

var marshallers = []MarshallerTestCase[model.EngineSpec]{
	{
		Name:        "yaml",
		Marshaller:  model.YAMLMarshalWithMax[model.EngineSpec],
		Unmarshaler: model.YAMLUnmarshalWithMax[model.EngineSpec],
	},
	{
		Name:        "json",
		Marshaller:  model.JSONMarshalWithMax[model.EngineSpec],
		Unmarshaler: model.JSONUnmarshalWithMax[model.EngineSpec],
	},
}

func TestRemoteRoundTrip(t *testing.T) {
	entryModule := model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		Name:          "TEST_IPFS",
		CID:           "doesn't matter",
	}
	entrypoint := "_start"
	parameters := []string{"1", "2", "3"}
	envvars := map[string]string{
		"FOO": "BAR",
	}
	importModules := []model.StorageSpec{entryModule, entryModule}

	t.Run("happy path", func(t *testing.T) {
		for _, er := range marshallers {
			t.Run(er.Name, func(t *testing.T) {

				clientEngineSpec := wasm.NewEngineSpec(entryModule, entrypoint, parameters, envvars, importModules)

				// simulate an API call from client to server
				engineBytes, err := er.Marshaller(clientEngineSpec)
				require.NoError(t, err)

				var serverEngineSpec model.EngineSpec
				err = er.Unmarshaler(engineBytes, &serverEngineSpec)
				require.NoError(t, err)

				wasmEngine, err := wasm.AsEngine(serverEngineSpec)
				require.NoError(t, err)

				assert.Equal(t, entryModule, wasmEngine.EntryModule)
				assert.Equal(t, entrypoint, wasmEngine.Entrypoint)
				assert.Equal(t, parameters, wasmEngine.Parameters)
				assert.Equal(t, envvars, wasmEngine.EnvironmentVariables)
				assert.Equal(t, importModules, wasmEngine.ImportModules)
			})
		}
	})
}
