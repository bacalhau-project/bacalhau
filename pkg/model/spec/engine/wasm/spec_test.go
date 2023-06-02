package wasm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	spec "github.com/bacalhau-project/bacalhau/pkg/model/spec"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
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
	// TODO add test cases for name and mount
	s3EntryModule, err := s3Spec.AsSpec("name", "mount")
	require.NoError(t, err)
	s3EntryModule.Metadata.Put("foo", "bar")

	expectedEngine := wasm.WasmEngineSpec{
		EntryModule:          &s3EntryModule,
		EntryPoint:           "entry",
		Parameters:           []string{"one", "two"},
		EnvironmentVariables: []string{"foo", "bar"},
		ImportModules:        []spec.Storage{s3EntryModule},
	}

	es, err := expectedEngine.AsSpec()
	require.NoError(t, err)

	require.NotEmpty(t, es.SchemaData)
	require.NotEmpty(t, es.Params)

	require.True(t, wasm.EngineSchema.Cid().Equals(es.Schema))

	t.Log(string(es.SchemaData))
	t.Log(string(es.Params))

	actualEngine, err := wasm.Decode(es)
	require.NoError(t, err)

	engineCid, err := actualEngine.Cid()
	require.NoError(t, err)
	t.Log(engineCid.String())

	assert.True(t, s3.StorageType.Equals(actualEngine.EntryModule.Schema))

	assert.Equal(t, expectedEngine.EntryModule, actualEngine.EntryModule)
	assert.Equal(t, expectedEngine.EntryPoint, actualEngine.EntryPoint)
	assert.Equal(t, expectedEngine.Parameters, actualEngine.Parameters)
	assert.Equal(t, expectedEngine.EnvironmentVariables, actualEngine.EnvironmentVariables)
	assert.Equal(t, expectedEngine.ImportModules, actualEngine.ImportModules)

}

func TestEmpty(t *testing.T) {
	s3Spec := &s3.S3StorageSpec{
		Bucket:         "bucket",
		Key:            "key",
		ChecksumSHA256: "checksum",
		VersionID:      "versionID",
		Endpoint:       "endpoint",
		Region:         "region",
	}
	// TODO add test cases for name and mount
	s3EntryModule, err := s3Spec.AsSpec("name", "mount")
	require.NoError(t, err)
	s3EntryModule.Metadata.Put("foo", "bar")

	e := enginetesting.WasmMakeEngine(t)
	require.NotNil(t, e)
}
