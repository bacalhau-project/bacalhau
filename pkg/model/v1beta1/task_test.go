package v1beta1

import (
	"embed"
	"path"
	"testing"

	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/stretchr/testify/require"
)

//go:embed tasks/*.json
var tests embed.FS

func TestCanUnmarshal(t *testing.T) {
	entries, err := tests.ReadDir("tasks")
	require.NoError(t, err)

	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			bytes, err := tests.ReadFile(path.Join("tasks", entry.Name()))
			require.NoError(t, err)

			task, err := UnmarshalIPLD[Task](bytes, json.Decode, UCANTaskSchema)
			require.NoError(t, err)

			_, err = task.ToSpec()
			require.NoError(t, err)
		})
	}
}

func TestConfig(t *testing.T) {
	bytes, err := tests.ReadFile("tasks/task_with_config.json")
	require.NoError(t, err)

	task, err := UnmarshalIPLD[Task](bytes, json.Decode, UCANTaskSchema)
	require.NoError(t, err)

	spec, err := task.ToSpec()
	require.NoError(t, err)

	require.Equal(t, VerifierNoop, spec.Verifier)
	require.Equal(t, PublisherIpfs, spec.Publisher)
	require.Equal(t, []string{"hello"}, spec.Annotations)
	require.Equal(t, "1m", spec.Resources.CPU)
	require.Equal(t, "1GB", spec.Resources.Disk)
	require.Equal(t, "1GB", spec.Resources.Memory)
	require.Equal(t, "0", spec.Resources.GPU)
	require.Equal(t, 300.0, spec.Timeout)
	require.Equal(t, false, spec.DoNotTrack)
}
