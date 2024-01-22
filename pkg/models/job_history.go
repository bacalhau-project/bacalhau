//go:generate stringer -type=JobHistoryType --trimprefix=JobHistoryType --output job_history_string.go
package models

import (
	"strings"
	"time"
)

type JobHistoryType int

const (
	JobHistoryTypeUndefined JobHistoryType = iota
	JobHistoryTypeJobLevel
	JobHistoryTypeExecutionLevel
)

func (s JobHistoryType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *JobHistoryType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	for typ := JobHistoryTypeUndefined; typ <= JobHistoryTypeExecutionLevel; typ++ {
		if strings.EqualFold(typ.String(), strings.TrimSpace(name)) {
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
// at the job level, or execution (node) level.
//
// {Job,Event}State fields will only be present if the Type field is of
// the matching type.
type JobHistory struct {
	Type           JobHistoryType                   `json:"Type"`
	JobID          string                           `json:"JobID"`
	NodeID         string                           `json:"NodeID,omitempty"`
	ExecutionID    string                           `json:"ExecutionID,omitempty"`
	JobState       *StateChange[JobStateType]       `json:"JobState,omitempty"`
	ExecutionState *StateChange[ExecutionStateType] `json:"ExecutionState,omitempty"`
	NewRevision    uint64                           `json:"NewRevision"`
	Comment        string                           `json:"Comment,omitempty"`
	Time           time.Time                        `json:"Time"`
}
