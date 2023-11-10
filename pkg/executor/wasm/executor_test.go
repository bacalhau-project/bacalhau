package wasm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Mocking necessary dependencies and structures

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) active() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockHandler) run(ctx context.Context) {
	_ = m.Called(ctx)
}

type mockExecutor struct {
	mock.Mock
}

func (m *mockExecutor) Get(executionID string) (*mockHandler, bool) {
	args := m.Called(executionID)
	return args.Get(0).(*mockHandler), args.Bool(1)
}

func (m *mockExecutor) Put(executionID string, handler *mockHandler) {
	_ = m.Called(executionID, handler)
}

func TestFailingRequestedMemGreaterThan4GB(t *testing.T) {
	e, err := NewExecutor()

	assert.Nil(t, err)

	resourcesConfig, err := models.NewResourcesConfigBuilder().
		Memory("4GB").
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
			Params: ,
		},
	}

	err = e.Start(context.Background(), r)

	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "requested memory exceeds the wasm limit")
}
