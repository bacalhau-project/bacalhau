//go:build unit || !integration

package store

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateNewExecution(t *testing.T) {
	execution := newExecution()
	err := ValidateNewExecution(execution)
	assert.NoError(t, err)
}

func TestValidateNewExecution_InvalidState(t *testing.T) {
	execution := newExecution()
	execution.State = ExecutionStateBidAccepted
	err := ValidateNewExecution(execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionState{})
}

func TestValidateNewExecution_InvalidVersion(t *testing.T) {
	execution := newExecution()
	execution.Version = 2
	err := ValidateNewExecution(execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionVersion{})
}

func newExecution() Execution {
	return *NewExecution(
		uuid.NewString(),
		model.Job{
			Metadata: model.Metadata{
				ID: uuid.NewString(),
			},
		},
		"nodeID-1",
		model.ResourceUsageData{
			CPU:    1,
			Memory: 2,
		})
}
