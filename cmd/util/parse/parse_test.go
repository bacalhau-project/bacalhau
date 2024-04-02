//go:build unit || !integration

package parse

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func outputsContain(specs []model.StorageSpec, name, path string) assert.Comparison {
	return func() (success bool) {
		for _, spec := range specs {
			if spec.Name == name && spec.Path == path {
				return true
			}
		}
		return false
	}
}

func TestJobOutputsAddsDefault(t *testing.T) {
	specs, err := JobOutputs(context.Background(), []string{})
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, DefaultOutputSpec, specs[0])
}

func TestJobOutputDoesNotAddDefaultTwice(t *testing.T) {
	specs, err := JobOutputs(context.Background(), []string{"test:" + DefaultOutputSpec.Path})
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, "test", specs[0].Name)
	require.Equal(t, DefaultOutputSpec.Path, specs[0].Path)
}

func TestJobOutputsDoesNotAddPathTwice(t *testing.T) {
	specs, err := JobOutputs(context.Background(), []string{"a:/a", "b:/a"})
	require.NoError(t, err)
	require.Len(t, specs, 2)
	require.Contains(t, specs, DefaultOutputSpec)
	require.Condition(t, outputsContain(specs, "b", "/a"))
}

func TestJobOutputsCreatesCorrectSpec(t *testing.T) {
	specs, err := JobOutputs(context.Background(), []string{"something:/else"})
	require.NoError(t, err)
	require.Len(t, specs, 2)
	require.Contains(t, specs, DefaultOutputSpec)
	require.Condition(t, outputsContain(specs, "something", "/else"))
}

func TestJobOutputsRejectsInvalidSpecs(t *testing.T) {
	_, err := JobOutputs(context.Background(), []string{"invalid"})
	require.Error(t, err)
}
