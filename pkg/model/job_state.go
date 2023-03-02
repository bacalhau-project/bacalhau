package model

import (
	"time"
)

// JobStateType The state of a job across the whole network that represents an aggregate view across
// the shards and nodes.
//
//go:generate stringer -type=JobStateType --trimprefix=JobState --output job_state_string.go
type JobStateType int

// these are the states a job can be in against a single node
const (
	JobStateNew JobStateType = iota // must be first

	JobStateInProgress

	// Job is canceled by the user.
	JobStateCancelled

	// All shards have failed
	JobStateError

	// Some shards have failed, while others have completed successfully
	JobStatePartialError

	// All shards have completed successfully
	JobStateCompleted
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
// the shards and nodes.
type JobState struct {
	// JobID is the unique identifier for the job
	JobID string `json:"JobID"`
	// Shards is a map of shard index to shard state.
	// The number of shards are fixed at the time of job creation.
	Shards map[int]ShardState `json:"Shards"`
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
