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

// JobHistory represents a single event in the history of a job.
// An event can be at the job level, shard level, or execution (node) level.
type JobHistory struct {
	Type             JobHistoryType     `json:"Type"`
	JobID            string             `json:"JobID"`
	ShardIndex       int                `json:"ShardIndex,omitempty"`
	NodeID           string             `json:"NodeID,omitempty"`
	ComputeReference string             `json:"ComputeReference,omitempty"`
	PreviousState    string             `json:"PreviousState"`
	NewState         string             `json:"NewState"`
	NewStateType     ExecutionStateType `json:"NewStateType,omitempty"` // only present for execution level events
	NewVersion       int                `json:"NewVersion"`
	Comment          string             `json:"Comment,omitempty"`
	Time             time.Time          `json:"Time"`
}
