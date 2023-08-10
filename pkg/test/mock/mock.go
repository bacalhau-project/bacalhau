package mock

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/google/uuid"
)

func Eval() *models.Evaluation {
	now := time.Now().UTC().UnixNano()
	eval := &models.Evaluation{
		ID:         uuid.NewString(),
		Namespace:  models.DefaultNamespace,
		Priority:   50,
		Type:       models.JobTypeBatch,
		JobID:      uuid.NewString(),
		Status:     models.EvalStatusPending,
		CreateTime: now,
		ModifyTime: now,
	}
	return eval
}

func Job() *models.Job {
	return &models.Job{
		ID:        uuid.NewString(),
		Name:      "test-job",
		Type:      models.JobTypeBatch,
		Namespace: models.DefaultNamespace,
		Count:     1,
		State: models.State[models.JobStateType]{
			StateType: models.JobStateTypeRunning,
		},
		Version:    1,
		Revision:   2,
		CreateTime: time.Now().UTC().UnixNano(),
		ModifyTime: time.Now().UTC().UnixNano(),
		Tasks: []*models.Task{
			{
				Name: "task1",
				Engine: &models.SpecConfig{
					Type: "noop",
				},
				Publisher: &models.SpecConfig{
					Type: "noop",
				},
			},
		},
	}
}

func Execution(job *models.Job) *models.Execution {
	now := time.Now().UTC()
	return &models.Execution{
		JobID:  job.ID,
		Job:    job,
		NodeID: uuid.NewString(),
		ID:     uuid.NewString(),
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateBidAccepted,
		},
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateRunning,
		},
		Revision:   4,
		CreateTime: now.UnixNano(),
		ModifyTime: now.UnixNano(),
	}
}

func Executions(job *models.Job, executionCount int) []*models.Execution {
	executions := make([]*models.Execution, executionCount)
	for i := 0; i < executionCount; i++ {
		executions[i] = Execution(job)
	}
	return executions
}

func Plan() *models.Plan {
	job := Job()
	eval := Eval()
	eval.JobID = job.ID

	return &models.Plan{
		EvalID:            eval.ID,
		EvalReceipt:       uuid.NewString(),
		Eval:              eval,
		Priority:          50,
		Job:               job,
		JobStateRevision:  1,
		NewExecutions:     []*models.Execution{},
		UpdatedExecutions: make(map[string]*models.PlanExecutionDesiredUpdate),
	}
}
