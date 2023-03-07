package model

import (
	"time"
)

// JobStateType The state of a job across the whole network that represents an aggregate view across
// the executions and nodes.
//
//go:generate stringer -type=JobStateType --trimprefix=JobState --output job_state_string.go
type JobStateType int

// these are the states a job can be in against a single node
const (
	JobStateNew JobStateType = iota // must be first

	JobStateInProgress

	// Job is canceled by the user.
	JobStateCancelled

	// Job have failed
	JobStateError

	// Job completed successfully
	JobStateCompleted

	// Job is waiting to be scheduled.
	JobStateQueued
)

// IsTerminal returns true if the given job type signals the end of the lifecycle of
// that job and that no change in the state can be expected.
func (s JobStateType) IsTerminal() bool {
	return s == JobStateCompleted || s == JobStateError || s == JobStateCancelled
}

func (s JobStateType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *JobStateType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	for typ := JobStateNew; typ <= JobStateCompleted; typ++ {
		if equal(typ.String(), name) {
			*s = typ
			return
		}
	}
	return
}

// JobState The state of a job across the whole network that represents an aggregate view across
// the executions and nodes.
type JobState struct {
	// JobID is the unique identifier for the job
	JobID string `json:"JobID"`
	// Executions is a list of executions of the job across the nodes.
	// A new execution is created when a node is selected to execute the job, and a node can have multiple executions for the same
	// job due to retries, but there can only be a single active execution per node at any given time.
	Executions []ExecutionState `json:"Executions"`
	// State is the current state of the job
	State JobStateType `json:"State"`
	// Version is the version of the job state. It is incremented every time the job state is updated.
	Version int `json:"Version"`
	// CreateTime is the time when the job was created.
	CreateTime time.Time `json:"CreateTime"`
	// UpdateTime is the time when the job state was last updated.
	UpdateTime time.Time `json:"UpdateTime"`
	// TimeoutAt is the time when the job will be timed out if it is not completed.
	TimeoutAt time.Time `json:"TimeoutAt,omitempty"`
}

func (j JobState) ExecutionsInTerminalState() bool {
	for _, execution := range j.Executions {
		if !execution.State.IsTerminal() {
			return false
		}
	}

	return true
}
