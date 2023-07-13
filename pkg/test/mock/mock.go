package mock

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/google/uuid"
)

func Eval() *models.Evaluation {
	now := time.Now().UTC().UnixNano()
	eval := &models.Evaluation{
		ID:         uuid.NewString(),
		Namespace:  model.DefaultNamespace,
		Priority:   50,
		Type:       model.JobTypeBatch,
		JobID:      uuid.NewString(),
		Status:     models.EvalStatusPending,
		CreateTime: now,
		ModifyTime: now,
	}
	return eval
}

func Job() *model.Job {
	return &model.Job{
		APIVersion: model.APIVersionLatest().String(),
		Metadata: model.Metadata{
			ID:        uuid.NewString(),
			CreatedAt: time.Now().UTC(),
		},
		Spec: model.Spec{
			Engine: model.EngineDocker,
			PublisherSpec: model.PublisherSpec{
				Type: model.PublisherNoop,
			},
			Deal: model.Deal{
				Concurrency: 1,
			},
		},
	}
}

func JobState(jobID string, executionCount int) *model.JobState {
	now := time.Now().UTC()
	executions := make([]model.ExecutionState, executionCount)
	for i := 0; i < executionCount; i++ {
		executions[i] = *ExecutionState(jobID)
	}
	return &model.JobState{
		JobID:      jobID,
		Executions: executions,
		State:      model.JobStateInProgress,
		CreateTime: now,
		UpdateTime: now,
	}
}

func ExecutionState(jobID string) *model.ExecutionState {
	now := time.Now().UTC()
	return &model.ExecutionState{
		JobID:            jobID,
		NodeID:           uuid.NewString(),
		ComputeReference: uuid.NewString(),
		State:            model.ExecutionStateBidAccepted,
		Version:          4,
		CreateTime:       now,
		UpdateTime:       now,
	}
}
