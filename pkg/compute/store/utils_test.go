//go:build unit || !integration

package store

import (
	"testing"

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
	execution.State = ExecutionStateRunning
	err := ValidateNewExecution(execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionState{})
}

func TestValidateNewExecution_InvalidVersion(t *testing.T) {
	execution := newExecution()
	execution.Version = 2
	err := ValidateNewExecution(execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionVersion{})
}

func newExecution() LocalState {
	return *NewLocalState(
		uuid.NewString(),
		models.Job{
			Metadata: models.Metadata{
				ID: uuid.NewString(),
			},
		},
		"nodeID-1",
		models.Resources{
			CPU:    1,
			Memory: 2,
		})
}
