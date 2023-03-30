package database

import (
	"time"

	"github.com/jackc/pgtype"
)

type Job struct {
	JobID string
	// model.Job
	Job pgtype.JSONB `gorm:"type:jsonb;default:'[]';not null"`

	CreatedAt time.Time
}

type JobState struct {
	JobID         string
	Version       int
	CurrentState  int
	PreviousState int

	CreatedAt time.Time
}

type JobExecution struct {
	JobID       string
	ExecutionID string
}

type NodeExecution struct {
	NodeID      string
	ExecutionID string
}

type ExecutionState struct {
	JobID            string
	NodeID           string
	ComputeReference string
	Status           string
	Version          int
	CurrentState     int
	PreviousState    int

	CreatedAt time.Time
}

type ExecutionOutput struct {
	ExecutionID string
	Version     int
	// model.RunCommandResult
	Output pgtype.JSONB `gorm:"type:jsonb;default:'[]';not null"`

	CreatedAt time.Time
}

type ExecutionVerificationProposal struct {
	ExecutionID string
	Version     int
	Proposal    []byte

	CreatedAt time.Time
}

type ExecutionVerificationResult struct {
	ExecutionID string
	Version     int
	Complete    bool
	Result      bool

	CreatedAt time.Time
}

type ExecutionPublishResult struct {
	ExecutionID string
	Version     int
	// model.StorageSpec
	Result pgtype.JSONB `gorm:"type:jsonb;default:'[]';not null"`

	CreatedAt time.Time
}
