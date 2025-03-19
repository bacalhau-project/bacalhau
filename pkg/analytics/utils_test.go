package analytics

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (s *UtilsTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *UtilsTestSuite) TestHashString() {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple string",
			input:    "test",
			expected: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		},
		{
			name:     "complex string",
			input:    "hello world",
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := hashString(tc.input)
			s.Equal(tc.expected, result)
		})
	}
}

func (s *UtilsTestSuite) TestGetDockerImageTelemetry() {
	testCases := []struct {
		name     string
		engine   *models.SpecConfig
		expected string
	}{
		{
			name:     "nil engine",
			engine:   nil,
			expected: "",
		},
		{
			name: "non-docker engine",
			engine: &models.SpecConfig{
				Type: "wasm",
			},
			expected: "",
		},
		{
			name: "docker engine with no image",
			engine: &models.SpecConfig{
				Type:   models.EngineDocker,
				Params: map[string]interface{}{},
			},
			expected: "",
		},
		{
			name: "trusted bacalhau image with Image key",
			engine: &models.SpecConfig{
				Type: models.EngineDocker,
				Params: map[string]interface{}{
					"Image": "ghcr.io/bacalhau-project/test:latest",
				},
			},
			expected: "ghcr.io/bacalhau-project/test:latest",
		},
		{
			name: "trusted bacalhau image with image key",
			engine: &models.SpecConfig{
				Type: models.EngineDocker,
				Params: map[string]interface{}{
					"image": "ghcr.io/bacalhau-project/test:latest",
				},
			},
			expected: "ghcr.io/bacalhau-project/test:latest",
		},
		{
			name: "trusted expanso image with Image key",
			engine: &models.SpecConfig{
				Type: models.EngineDocker,
				Params: map[string]interface{}{
					"Image": "expanso/test:latest",
				},
			},
			expected: "expanso/test:latest",
		},
		{
			name: "trusted expanso image with image key",
			engine: &models.SpecConfig{
				Type: models.EngineDocker,
				Params: map[string]interface{}{
					"image": "expanso/test:latest",
				},
			},
			expected: "expanso/test:latest",
		},
		{
			name: "non-trusted image with Image key",
			engine: &models.SpecConfig{
				Type: models.EngineDocker,
				Params: map[string]interface{}{
					"Image": "docker.io/random/image:latest",
				},
			},
			expected: hashString("docker.io/random/image:latest"),
		},
		{
			name: "non-trusted image with image key",
			engine: &models.SpecConfig{
				Type: models.EngineDocker,
				Params: map[string]interface{}{
					"image": "docker.io/random/image:latest",
				},
			},
			expected: hashString("docker.io/random/image:latest"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetDockerImageTelemetry(tc.engine)
			s.Equal(tc.expected, result)
		})
	}
}
