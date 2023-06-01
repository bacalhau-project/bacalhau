//go:build unit || !integration

package docker_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DockerCacheTestSuite struct {
	suite.Suite
}

func TestDockerCache(t *testing.T) {
	suite.Run(t, new(DockerCacheTestSuite))
}

func (s *DockerCacheTestSuite) withEnvVars(settings map[string]string) (closer func()) {
	original := map[string]string{}

	for name, value := range settings {
		if originalValue, ok := os.LookupEnv(name); ok {
			original[name] = originalValue
		}
		_ = os.Setenv(name, value)
	}

	return func() {
		for key := range settings {
			val, present := original[key]
			if present {
				_ = os.Setenv(key, val)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}
}

func (s *DockerCacheTestSuite) TestDefaultsSize() {
	env := map[string]string{
		"DOCKER_MANIFEST_CACHE_SIZE": "10",
	}
	cleanup := s.withEnvVars(env)
	s.T().Cleanup(cleanup)

	sampleVal := docker.ImageManifest{}

	mc := docker.InitManifestCache()
	for i := 0; i < 10; i++ {
		err := mc.Set(fmt.Sprintf("k%d", i), sampleVal, 1, 10)
		require.NoError(s.T(), err)
	}

	err := mc.Set("b", sampleVal, 1, 10)
	require.Error(s.T(), err) // too costly to write the 11th value
}
