package v1beta1

import (
	"testing"

	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalWasm(t *testing.T) {
	bytes, err := tests.ReadFile("tasks/wasm_task.json")
	require.NoError(t, err)

	task, err := UnmarshalIPLD[Task](bytes, json.Decode, UCANTaskSchema)
	require.NoError(t, err)

	spec, err := task.ToSpec()
	require.NoError(t, err)
	require.Equal(t, EngineWasm, spec.Engine)
	require.Equal(t, "_start", spec.Wasm.EntryPoint)
	require.Equal(t, []string{"/inputs/data.tar.gz"}, spec.Wasm.Parameters)
	require.Equal(t, map[string]string{"HELLO": "world"}, spec.Wasm.EnvironmentVariables)
	require.Equal(t, []StorageSpec{
		{Path: "/job", StorageSource: StorageSourceIPFS, CID: "bafybeig7mdkzcgpacpozamv7yhhaelztfrnb6ozsupqqh7e5uyqdkijegi"},
	}, spec.Contexts)
	require.Equal(t, []StorageSpec{
		{Path: "/inputs", StorageSource: StorageSourceURLDownload, URL: "https://www.example.com/data.tar.gz"},
	}, spec.Inputs)
	require.Equal(t, []StorageSpec{
		{Path: "/outputs", Name: "outputs"},
	}, spec.Outputs)
}
