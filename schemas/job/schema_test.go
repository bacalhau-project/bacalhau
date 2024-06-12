//go:build unit || !integration

package job

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test_vectors/valid/*
var validVectors embed.FS

//go:embed test_vectors/invalid/*
var invalidVectors embed.FS

func TestValidJobs(t *testing.T) {
	jobSchema, err := Schema()
	require.NoError(t, err)

	entries, err := validVectors.ReadDir("test_vectors/valid")
	require.NoError(t, err)
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {

			// Open the file with the correct relative path
			vf, err := validVectors.Open("test_vectors/valid/" + entry.Name())
			require.NoError(t, err)
			defer vf.Close() // Ensure the file is closed after the test

			result, err := jobSchema.ValidateReader(vf)
			require.NoError(t, err)
			require.True(t, result.Valid(), result.Errors())
		})
	}
}

func TestInvalidJobs(t *testing.T) {
	jobSchema, err := Schema()
	require.NoError(t, err)

	entries, err := invalidVectors.ReadDir("test_vectors/invalid")
	require.NoError(t, err)
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {

			// Open the file with the correct relative path
			vf, err := invalidVectors.Open("test_vectors/invalid/" + entry.Name())
			require.NoError(t, err)
			defer vf.Close() // Ensure the file is closed after the test

			result, err := jobSchema.ValidateReader(vf)
			require.NoError(t, err)
			require.False(t, result.Valid())
		})
	}
}
