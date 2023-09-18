package parse

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobOutputsDoesNotAddDefaults(t *testing.T) {
	specs, err := JobOutputs(context.Background(), []string{})
	require.NoError(t, err)
	require.Empty(t, specs)
}

func TestJobOutputsCreatesCorrectSpec(t *testing.T) {
	specs, err := JobOutputs(context.Background(), []string{"outputs:/outputs"})
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, "outputs", specs[0].Name)
	require.Equal(t, "/outputs", specs[0].Path)
}
