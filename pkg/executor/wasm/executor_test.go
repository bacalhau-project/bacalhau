//go:build unit || !integration

package wasm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ExecutorTestSuite struct {
	suite.Suite
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}

func (s *ExecutorTestSuite) TestFailingRequestedMemGreaterThan4GB() {
	e, err := NewExecutor()

	s.Require().NoError(err)

	resourcesConfig, err := models.NewResourcesConfigBuilder().
		Memory("5GB").
		Build()

	s.Require().NoError(err)

	resources, err := resourcesConfig.ToResources()
	s.Require().NoError(err)

	r := &executor.RunCommandRequest{
		JobID:       "1",
		ExecutionID: "1",
		Resources:   resources,
		EngineParams: &models.SpecConfig{
			Type:   models.EngineWasm,
			Params: map[string]any{},
		},
	}

	err = e.Start(context.Background(), r)

	s.Require().Error(err)

	assert.Contains(s.T(), err.Error(), "requested memory exceeds the wasm limit")
}
