package inline_test

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	strgspec "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
)

func TestRoundTrip(t *testing.T) {
	expectedInline := inline.InlineStorageSpec{
		URL: "https://example.com",
	}

	expectedSpec, err := expectedInline.AsSpec()
	require.NoError(t, err)

	assert.Equal(t, "InlineStorageSpec", expectedSpec.Type)
	assert.Equal(t, inline.Schema.Cid(), expectedSpec.Schema)
	assert.NotEmpty(t, expectedSpec.SchemaData)
	assert.NotEmpty(t, expectedSpec.Params)

	actualInline, err := inline.Decode(expectedSpec)
	require.NoError(t, err)

	assert.Equal(t, expectedInline.URL, actualInline.URL)
}

func TestInvalidDecode(t *testing.T) {
	invalidSpec := strgspec.Spec{
		Type:       "Invalid",
		Schema:     cid.Undef,
		SchemaData: []byte{1, 2, 3, 4},
		Params:     []byte{1, 2, 3, 4, 5},
	}

	_, err := inline.Decode(invalidSpec)
	require.Error(t, err)

	invalidSpec = strgspec.Spec{
		Type:       "Invalid",
		Schema:     inline.Schema.Cid(),
		SchemaData: []byte{1, 2, 3, 4},
		Params:     []byte{1, 2, 3, 4, 5},
	}
	_, err = inline.Decode(invalidSpec)
	require.Error(t, err)
}
