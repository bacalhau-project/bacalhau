//go:build unit || !integration

package repo

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/assert" // Consider using this package for more readable assertions
)

func TestDecodeSpec_Git(t *testing.T) {
	spec := models.SpecConfig{
		Type: models.StorageSourceRepoClone,
		Params: map[string]interface{}{
			"Repo": "git://github.com/example/repo.git",
		},
	}

	source, err := DecodeSpec(&spec)
	assert.NoError(t, err)
	assert.Equal(t, "git://github.com/example/repo.git", source.Repo)
	assert.NoError(t, source.Validate())
}

func TestDecodeSpec_GitLFS(t *testing.T) {
	spec := models.SpecConfig{
		Type: models.StorageSourceRepoCloneLFS,
		Params: map[string]interface{}{
			"Repo": "gitlfs://github.com/example/repo.git",
		},
	}

	source, err := DecodeSpec(&spec)
	assert.NoError(t, err)
	assert.Equal(t, "gitlfs://github.com/example/repo.git", source.Repo)
	assert.NoError(t, source.Validate())
}

func TestDecodeSpec_UnsupportedType(t *testing.T) {
	spec := models.SpecConfig{
		Type: "UnsupportedType",
		Params: map[string]interface{}{
			"Repo": "git://github.com/example/repo.git",
		},
	}

	_, err := DecodeSpec(&spec)
	assert.Error(t, err)
}

func TestDecodeSpec_EmptyRepo(t *testing.T) {
	spec := models.SpecConfig{
		Type: models.StorageSourceRepoClone,
		Params: map[string]interface{}{
			"Repo": "",
		},
	}

	_, err := DecodeSpec(&spec)
	assert.Error(t, err)
}

func TestDecodeSpec_EmptySpec(t *testing.T) {
	spec := models.SpecConfig{
		Type: models.StorageSourceRepoClone,
	}

	_, err := DecodeSpec(&spec)
	assert.Error(t, err)
}
