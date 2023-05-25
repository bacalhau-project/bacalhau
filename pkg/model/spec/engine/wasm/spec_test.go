package wasm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
)

func TestRoundTrip(t *testing.T) {
	s3Spec := &s3.S3StorageSpec{
		Bucket:         "bucket",
		Key:            "key",
		ChecksumSHA256: "checksum",
		VersionID:      "versionID",
		Endpoint:       "endpoint",
		Region:         "region",
	}
	s3EntryModule, err := s3Spec.AsSpec()
	require.NoError(t, err)

	expectedEngine := wasm.WasmEngineSpec{
		EntryModule:          s3EntryModule,
		EntryPoint:           "entry",
		Parameters:           []string{"one", "two"},
		EnvironmentVariables: []string{"foo", "bar"},
		ImportModules:        []storage.Spec{s3EntryModule},
	}

	spec, err := expectedEngine.AsSpec()
	require.NoError(t, err)

	require.NotEmpty(t, spec.SchemaData)
	require.NotEmpty(t, spec.Params)

	require.True(t, wasm.EngineSchema.Cid().Equals(spec.Schema))

	t.Log(string(spec.SchemaData))
	t.Log(string(spec.Params))

	actualEngine, err := wasm.Decode(spec)
	require.NoError(t, err)

	engineCid, err := actualEngine.Cid()
	require.NoError(t, err)
	t.Log(engineCid.String())

	assert.True(t, s3.Schema.Cid().Equals(actualEngine.EntryModule.Schema))

	assert.Equal(t, expectedEngine.EntryModule, actualEngine.EntryModule)
	assert.Equal(t, expectedEngine.EntryPoint, actualEngine.EntryPoint)
	assert.Equal(t, expectedEngine.Parameters, actualEngine.Parameters)
	assert.Equal(t, expectedEngine.EnvironmentVariables, actualEngine.EnvironmentVariables)
	assert.Equal(t, expectedEngine.ImportModules, actualEngine.ImportModules)

}
