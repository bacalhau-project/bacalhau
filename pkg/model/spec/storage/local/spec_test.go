package local_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
)

func TestRoundTrip(t *testing.T) {
	expectedSpec := local.LocalStorageSpec{
		Source: "/path/to/local/data",
	}

	spec, err := expectedSpec.AsSpec("name", "mount")
	require.NoError(t, err)

	require.NotEmpty(t, spec.SchemaData)
	assert.Equal(t, "name", spec.Name)
	assert.Equal(t, "mount", spec.Mount)
	require.NotEmpty(t, spec.Params)
	require.True(t, local.Schema.Cid().Equals(spec.Schema))

	t.Log(string(spec.SchemaData))
	t.Log(string(spec.Params))

	actualSpec, err := local.Decode(spec)
	require.NoError(t, err)

	assert.Equal(t, expectedSpec.Source, actualSpec.Source)

}
