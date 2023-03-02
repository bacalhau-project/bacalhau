package model

import (
	"time"
)

type JobHistoryType int

const (
	jobHistoryTypeUndefined JobHistoryType = iota
	JobHistoryTypeJobLevel
	JobHistoryTypeShardLevel
	JobHistoryTypeExecutionLevel
)

func (s JobHistoryType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *JobHistoryType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	for typ := jobHistoryTypeUndefined; typ <= JobHistoryTypeExecutionLevel; typ++ {
		if equal(typ.String(), name) {
			*s = typ
			return
		}
	}
	return
}

// StateChange represents a change in state of one of the state types.
type StateChange[StateType any] struct {
	Previous StateType `json:"Previous,omitempty"`
	New      StateType `json:"New,omitempty"`
}

// JobHistory represents a single event in the history of a job. An event can be
// at the job level, shard level, or execution (node) level.
//
// {Job,Shard,Event}State fields will only be present if the Type field is of
// the matching type.
type JobHistory struct {
	Type             JobHistoryType                   `json:"Type"`
	JobID            string                           `json:"JobID"`
	ShardIndex       int                              `json:"ShardIndex,omitempty"`
	NodeID           string                           `json:"NodeID,omitempty"`
	ComputeReference string                           `json:"ComputeReference,omitempty"`
	JobState         *StateChange[JobStateType]       `json:"JobState,omitempty"`
	ShardState       *StateChange[ShardStateType]     `json:"ShardState,omitempty"`
	ExecutionState   *StateChange[ExecutionStateType] `json:"ExecutionState,omitempty"`
	NewVersion       int                              `json:"NewVersion"`
	Comment          string                           `json:"Comment,omitempty"`
	Time             time.Time                        `json:"Time"`
}
