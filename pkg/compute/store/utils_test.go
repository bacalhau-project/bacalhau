//go:build unit || !integration

package store

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func TestValidateNewExecution(t *testing.T) {
	execution := mock.ExecutionForJob(mock.Job())
	err := ValidateNewExecution(execution)
	assert.NoError(t, err)
}

func TestValidateNewExecution_InvalidState(t *testing.T) {
	execution := mock.ExecutionForJob(mock.Job())
	execution.ComputeState.StateType = models.ExecutionStateRunning
	err := ValidateNewExecution(execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionState{})
}

func TestValidateNewExecution_InvalidRevision(t *testing.T) {
	execution := mock.ExecutionForJob(mock.Job())
	execution.Revision = 2
	err := ValidateNewExecution(execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionRevision{})
}
