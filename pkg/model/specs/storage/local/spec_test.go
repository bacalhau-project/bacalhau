package local_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/specs/storage/local"
)

func TestRoundTrip(t *testing.T) {
	expectedSpec := local.LocalStorageSpec{
		Path: "/path/to/local/data",
	}

	spec, err := expectedSpec.AsSpec()
	require.NoError(t, err)

	require.NotEmpty(t, spec.SchemaData)
	require.NotEmpty(t, spec.Params)
	require.True(t, local.StorageSchema.Cid().Equals(spec.Schema))

	t.Log(string(spec.SchemaData))
	t.Log(string(spec.Params))

	actualSpec, err := local.Decode(spec)
	require.NoError(t, err)

	assert.Equal(t, expectedSpec.Path, actualSpec.Path)

}
