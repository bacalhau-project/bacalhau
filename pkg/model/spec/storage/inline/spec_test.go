//go:build unit || !integration

package inline_test

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
)

func TestRoundTrip(t *testing.T) {
	expectedInline := inline.InlineStorageSpec{
		URL: "https://example.com",
	}

	expectedSpec, err := expectedInline.AsSpec("name", "mount")
	require.NoError(t, err)

	assert.Equal(t, "InlineStorageSpec", expectedSpec.Type)
	assert.Equal(t, inline.Schema.Cid(), expectedSpec.Schema)
	assert.Equal(t, "name", expectedSpec.Name)
	assert.Equal(t, "mount", expectedSpec.Mount)
	assert.NotEmpty(t, expectedSpec.SchemaData)
	assert.NotEmpty(t, expectedSpec.Params)

	actualInline, err := inline.Decode(expectedSpec)
	require.NoError(t, err)

	assert.Equal(t, expectedInline.URL, actualInline.URL)
}

func TestInvalidDecode(t *testing.T) {
	invalidSpec := spec.Storage{
		Type:       "Invalid",
		Name:       "name",
		Mount:      "mount",
		Schema:     cid.Undef,
		SchemaData: []byte{1, 2, 3, 4},
		Params:     []byte{1, 2, 3, 4, 5},
	}

	_, err := inline.Decode(invalidSpec)
	require.Error(t, err)

	invalidSpec = spec.Storage{
		Type:       "Invalid",
		Name:       "name",
		Mount:      "mount",
		Schema:     inline.Schema.Cid(),
		SchemaData: []byte{1, 2, 3, 4},
		Params:     []byte{1, 2, 3, 4, 5},
	}
	_, err = inline.Decode(invalidSpec)
	require.Error(t, err)
}
