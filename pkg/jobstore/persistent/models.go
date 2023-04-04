package persistent

import (
	"fmt"
	"time"

	"github.com/jackc/pgtype"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type Job struct {
	JobID string // PK
	// model.Job
	Job pgtype.JSONB `gorm:"type:jsonb;default:'[]';not null"`

	CreatedAt time.Time
}

func (j *Job) AsJob() (model.Job, error) {
	if j.Job.Status == pgtype.Null {
		return model.Job{}, fmt.Errorf("invalid Job model: model.Job is null")
	}
	var out model.Job
	if err := j.Job.AssignTo(&out); err != nil {
		return model.Job{}, err
	}
	return out, nil

}

type JobState struct {
	JobID   string // PK
	Version int    // PK

	CurrentState  int
	PreviousState int
	Comment       string

	CreatedAt time.Time
}

type ExecutionState struct {
	JobID            string // PK
	NodeID           string // PK
	ComputeReference string // PK
	Version          int    // PK

	CurrentState  int
	PreviousState int
	Comment       string

	CreatedAt time.Time

	Execution pgtype.JSONB `gorm:"type:jsonb;default:'[]';not null"`
}

func (e *ExecutionState) AsExecutionState() (model.ExecutionState, error) {
	if e.Execution.Status == pgtype.Null {
		return model.ExecutionState{}, fmt.Errorf("invalid ExecutionState mode: model.ExecutionState is null")
	}
	var out model.ExecutionState
	if err := e.Execution.AssignTo(&out); err != nil {
		return model.ExecutionState{}, err
	}
	out.JobID = e.JobID
	out.NodeID = e.NodeID
	out.ComputeReference = e.ComputeReference
	out.State = model.ExecutionStateType(e.CurrentState)
	out.Status = e.Comment
	out.Version = e.Version
	out.UpdateTime = e.CreatedAt
	return out, nil
}
