//go:build unit || !integration

package wasm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func TestFailingRequestedMemGreaterThan4GB(t *testing.T) {
	e, err := NewExecutor()

	assert.Nil(t, err)

	resourcesConfig, err := models.NewResourcesConfigBuilder().
		Memory("5GB").
		Build()

	assert.Nil(t, err)

	resources, err := resourcesConfig.ToResources()
	assert.Nil(t, err)

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

	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "requested memory exceeds the wasm limit")
}
