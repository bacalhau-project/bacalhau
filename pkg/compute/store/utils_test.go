//go:build unit || !integration

package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
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

func TestValidateNewExecution_InvalidRevision(t *testing.T) {
	execution := newExecution()
	execution.Revision = 2
	err := ValidateNewExecution(execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionRevision{})
}

func newExecution() LocalExecutionState {
	execution := mock.ExecutionForJob(mock.Job())
	return *NewLocalExecutionState(execution)
}
