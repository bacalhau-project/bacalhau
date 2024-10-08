//go:build unit || !integration

package docker_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	legacy_types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
)

type DockerCacheTestSuite struct {
	suite.Suite
}

func TestDockerCache(t *testing.T) {
	suite.Run(t, new(DockerCacheTestSuite))
}

func (s *DockerCacheTestSuite) TestDefaultsSize() {
	sampleVal := docker.ImageManifest{}

	mc := docker.NewManifestCache(types.DockerManifestCache{
		Size:    legacy_types.Testing.Node.Compute.ManifestCache.Size,
		TTL:     types.Duration(legacy_types.Testing.Node.Compute.ManifestCache.Duration),
		Refresh: types.Duration(legacy_types.Testing.Node.Compute.ManifestCache.Frequency),
	})
	for i := 0; i < 1000; i++ {
		err := mc.Set(fmt.Sprintf("k%d", i), sampleVal, 1, 10)
		require.NoError(s.T(), err)
	}

	err := mc.Set("b", sampleVal, 1, 10)
	require.Error(s.T(), err) // too costly to write the 1001st value
}
