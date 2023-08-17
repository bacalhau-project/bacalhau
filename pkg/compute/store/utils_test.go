//go:build unit || !integration

package store

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
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
	execution := mock.ExecutionForJob(mock.Job())
	return *NewLocalState(execution, "nodeID-1")
}
