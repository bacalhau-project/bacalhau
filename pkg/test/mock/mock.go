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
	job := &models.Job{
		ID:        uuid.NewString(),
		Name:      "test-job",
		Type:      models.JobTypeBatch,
		Namespace: models.DefaultNamespace,
		Count:     1,
		State: models.State[models.JobStateType]{
			StateType: models.JobStateTypePending,
		},
		Version: 1,
		Tasks:   []*models.Task{Task()},
	}
	job.Normalize()
	if err := job.Validate(); err != nil {
		panic(err)
	}
	return job
}

func Task() *models.Task {
	task := &models.Task{
		Name: "task1",
		Engine: &models.SpecConfig{
			Type: "Noop",
		},
		Publisher: &models.SpecConfig{
			Type: "Noop",
		},
		ResourcesConfig: &models.ResourcesConfig{
			CPU:    "0.1",
			Memory: "100Mi",
		},
		Network: &models.NetworkConfig{
			Type:    models.NetworkNone,
			Domains: make([]string, 0),
		},
		Timeouts: &models.TimeoutConfig{
			ExecutionTimeout: 30,
		},
	}
	task.Normalize()
	if err := task.Validate(); err != nil {
		panic(err)
	}
	return task
}

func TaskBuilder() *models.TaskBuilder {
	return models.NewTaskBuilderFromTask(Task())
}

func Execution() *models.Execution {
	return ExecutionForJob(Job())
}

func ExecutionForJob(job *models.Job) *models.Execution {
	execution := &models.Execution{
		JobID:  job.ID,
		Job:    job,
		NodeID: uuid.NewString(),
		ID:     uuid.NewString(),
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateNew,
		},
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStatePending,
		},
	}
	execution.Normalize()
	if err := execution.Validate(); err != nil {
		panic(err)
	}
	return execution
}

func Executions(job *models.Job, executionCount int) []*models.Execution {
	executions := make([]*models.Execution, executionCount)
	for i := 0; i < executionCount; i++ {
		executions[i] = ExecutionForJob(job)
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
		NewExecutions:     []*models.Execution{},
		UpdatedExecutions: make(map[string]*models.PlanExecutionDesiredUpdate),
	}
}
