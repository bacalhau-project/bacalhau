//go:build unit || !integration

package docker_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
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

	testConfig, err := config.NewTestConfig()
	s.Require().NoError(err)

	mc := docker.NewManifestCache(types.DockerManifestCache{
		Size:    testConfig.Engines.Types.Docker.ManifestCache.Size,
		TTL:     testConfig.Engines.Types.Docker.ManifestCache.TTL,
		Refresh: testConfig.Engines.Types.Docker.ManifestCache.Refresh,
	})
	for i := 0; i < 1000; i++ {
		err := mc.Set(fmt.Sprintf("k%d", i), sampleVal, 1, 10)
		require.NoError(s.T(), err)
	}

	err = mc.Set("b", sampleVal, 1, 10)
	require.Error(s.T(), err) // too costly to write the 1001st value
}
