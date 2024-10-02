//go:build unit || !integration

package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBacalhauCopy(t *testing.T) {
	t.Run("Copy of empty config", func(t *testing.T) {
		original := Bacalhau{}
		copied, err := original.Copy()
		require.NoError(t, err)
		assert.Equal(t, original, copied)
	})

	t.Run("Copy of config with all fields set", func(t *testing.T) {
		original := Bacalhau{
			API: API{
				Host: "localhost",
				Port: 8080,
				TLS: TLS{
					CertFile: "cert.pem",
					KeyFile:  "key.pem",
				},
			},
			NameProvider:       "test",
			DataDir:            "/data",
			StrictVersionMatch: true,
			Orchestrator: Orchestrator{
				Enabled: true,
				Host:    "orchestrator.local",
				Port:    4222,
			},
			// Set other fields...
		}
		copied, err := original.Copy()
		require.NoError(t, err)
		assert.Equal(t, original, copied)
	})

	t.Run("Modifying copy doesn't affect original", func(t *testing.T) {
		original := Bacalhau{
			API: API{
				Host: "localhost",
				Port: 8080,
			},
			NameProvider: "test",
		}
		copied, err := original.Copy()
		require.NoError(t, err)

		copied.API.Host = "modified"
		copied.NameProvider = "modified"

		assert.NotEqual(t, original, copied)
		assert.Equal(t, "localhost", original.API.Host)
		assert.Equal(t, "test", original.NameProvider)
	})

	t.Run("Deep copy of nested structs", func(t *testing.T) {
		original := Bacalhau{
			Orchestrator: Orchestrator{
				NodeManager: NodeManager{
					DisconnectTimeout: Duration(5),
				},
			},
		}
		copied, err := original.Copy()
		require.NoError(t, err)

		copied.Orchestrator.NodeManager.DisconnectTimeout = Duration(10)

		assert.NotEqual(t, original.Orchestrator.NodeManager, copied.Orchestrator.NodeManager)
	})

	t.Run("Copy of config with slices and maps", func(t *testing.T) {
		original := Bacalhau{
			Compute: Compute{
				Orchestrators: []string{"nats://127.0.0.1:4222", "nats://127.0.0.1:4223"},
			},
			// Assume we have a map field in one of the nested structs
			JobDefaults: JobDefaults{
				Batch: BatchJobDefaultsConfig{
					Task: BatchTaskDefaultConfig{
						Resources: ResourcesConfig{
							CPU:    "500m",
							Memory: "1Gb",
						},
					},
				},
			},
		}
		copied, err := original.Copy()
		require.NoError(t, err)

		assert.Equal(t, original, copied)

		// Modify the copy
		copied.Compute.Orchestrators[0] = "modified"
		copied.JobDefaults.Batch.Task.Resources.CPU = "1000m"

		assert.NotEqual(t, original, copied)
		assert.Equal(t, "nats://127.0.0.1:4222", original.Compute.Orchestrators[0])
		assert.Equal(t, "500m", original.JobDefaults.Batch.Task.Resources.CPU)
	})
}

func TestBacalhauMergeNew(t *testing.T) {
	t.Run("MergeNew with empty config", func(t *testing.T) {
		base := Bacalhau{
			API: API{
				Host: "localhost",
				Port: 8080,
			},
			NameProvider: "test",
		}
		other := Bacalhau{}
		MergeNewd, err := base.MergeNew(other)
		require.NoError(t, err)
		assert.Equal(t, base, MergeNewd)
	})

	t.Run("MergeNew overwrites existing fields", func(t *testing.T) {
		base := Bacalhau{
			API: API{
				Host: "localhost",
				Port: 8080,
			},
			NameProvider: "test",
		}
		other := Bacalhau{
			API: API{
				Host: "otherhost",
			},
			StrictVersionMatch: true,
		}
		MergeNewd, err := base.MergeNew(other)
		require.NoError(t, err)
		assert.Equal(t, "otherhost", MergeNewd.API.Host)
		assert.Equal(t, 8080, MergeNewd.API.Port)
		assert.Equal(t, "test", MergeNewd.NameProvider)
		assert.True(t, MergeNewd.StrictVersionMatch)
	})

	t.Run("MergeNew with nested structs", func(t *testing.T) {
		base := Bacalhau{
			Orchestrator: Orchestrator{
				Enabled: false,
				Host:    "base.local",
				NodeManager: NodeManager{
					DisconnectTimeout: Duration(5),
				},
			},
		}
		other := Bacalhau{
			Orchestrator: Orchestrator{
				Enabled: true,
				NodeManager: NodeManager{
					DisconnectTimeout: Duration(10),
				},
			},
		}
		MergeNewd, err := base.MergeNew(other)
		require.NoError(t, err)
		assert.True(t, MergeNewd.Orchestrator.Enabled)
		assert.Equal(t, "base.local", MergeNewd.Orchestrator.Host)
		assert.Equal(t, Duration(10), MergeNewd.Orchestrator.NodeManager.DisconnectTimeout)
	})

	t.Run("MergeNew with slices", func(t *testing.T) {
		base := Bacalhau{
			Compute: Compute{
				Orchestrators: []string{"nats://127.0.0.1:4222"},
			},
		}
		other := Bacalhau{
			Compute: Compute{
				Orchestrators: []string{"nats://127.0.0.1:4223", "nats://127.0.0.1:4224"},
			},
		}
		MergeNewd, err := base.MergeNew(other)
		require.NoError(t, err)
		assert.Equal(t, []string{"nats://127.0.0.1:4223", "nats://127.0.0.1:4224"}, MergeNewd.Compute.Orchestrators)
	})

	t.Run("MergeNew doesn't affect original configs", func(t *testing.T) {
		base := Bacalhau{
			API: API{
				Host: "localhost",
				Port: 8080,
			},
		}
		other := Bacalhau{
			API: API{
				Host: "otherhost",
			},
		}
		MergeNewd, err := base.MergeNew(other)
		require.NoError(t, err)

		assert.NotEqual(t, base, MergeNewd)
		assert.NotEqual(t, other, MergeNewd)
		assert.Equal(t, "localhost", base.API.Host)
		assert.Equal(t, "otherhost", other.API.Host)
	})

	t.Run("MergeNew with complex nested structures", func(t *testing.T) {
		base := Bacalhau{
			JobDefaults: JobDefaults{
				Batch: BatchJobDefaultsConfig{
					Priority: 0,
					Task: BatchTaskDefaultConfig{
						Resources: ResourcesConfig{
							CPU:    "500m",
							Memory: "1Gb",
						},
					},
				},
			},
		}
		other := Bacalhau{
			JobDefaults: JobDefaults{
				Batch: BatchJobDefaultsConfig{
					Priority: 1,
					Task: BatchTaskDefaultConfig{
						Resources: ResourcesConfig{
							CPU: "1000m",
						},
					},
				},
			},
		}
		MergeNewd, err := base.MergeNew(other)
		require.NoError(t, err)
		assert.Equal(t, 1, MergeNewd.JobDefaults.Batch.Priority)
		assert.Equal(t, "1000m", MergeNewd.JobDefaults.Batch.Task.Resources.CPU)
		assert.Equal(t, "1Gb", MergeNewd.JobDefaults.Batch.Task.Resources.Memory)
	})
}

// Helper function to compare Bacalhau structs deeply
func deepEqual(t *testing.T, expected, actual Bacalhau) {
	t.Helper()

	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)

	actualJSON, err := json.Marshal(actual)
	require.NoError(t, err)

	assert.JSONEq(t, string(expectedJSON), string(actualJSON))
}
