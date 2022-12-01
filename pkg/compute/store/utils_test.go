package store

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateNewExecution(t *testing.T) {
	execution := newExecution()
	err := ValidateNewExecution(context.Background(), execution)
	assert.NoError(t, err)
}

func TestValidateNewExecution_InvalidState(t *testing.T) {
	execution := newExecution()
	execution.State = ExecutionStateBidAccepted
	err := ValidateNewExecution(context.Background(), execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionState{})
}

func TestValidateNewExecution_InvalidVersion(t *testing.T) {
	execution := newExecution()
	execution.Version = 2
	err := ValidateNewExecution(context.Background(), execution)
	assert.ErrorAs(t, err, &ErrInvalidExecutionVersion{})
}

func newExecution() Execution {
	return *NewExecution(
		uuid.NewString(),
		model.JobShard{
			Job:   &model.Job{ID: uuid.NewString()},
			Index: 1,
		},
		model.ResourceUsageData{
			CPU:    1,
			Memory: 2,
		})
}
