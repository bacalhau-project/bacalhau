package telemetry

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDetachedContext_Value_valuesPassedThrough(t *testing.T) {
	expectedKey := "dummy"
	expectedValue := "value"

	ctx := NewDetachedContext(context.WithValue(context.Background(), expectedKey, expectedValue))

	actualValue := ctx.Value(expectedKey)

	assert.Equal(t, expectedValue, actualValue)
}

func TestDetachedContext_Deadline_separateFromParent(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx := NewDetachedContext(parentCtx)

	assert.NoError(t, parentCtx.Err())
	assert.NoError(t, ctx.Err())

	cancel()

	assert.Error(t, parentCtx.Err())
	assert.NoError(t, ctx.Err())
}
