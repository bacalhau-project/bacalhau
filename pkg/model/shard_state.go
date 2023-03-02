package model

import (
	"fmt"
	"time"
)

// ShardStateType represents the state of a shard in a job that represents an aggregate view across
// the nodes that are executing the shard.
type ShardStateType int

const (
	ShardStateNew ShardStateType = iota
	ShardStateInProgress
	// The job/shard is canceled by the user.
	ShardStateCancelled
	// The shard has failed due to an error.
	ShardStateError
	// The shard has been completed successfully.
	ShardStateCompleted
)

// IsTerminal returns true if the given shard state type signals the end of the execution of the shard.
func (s ShardStateType) IsTerminal() bool {
	return s == ShardStateCompleted || s == ShardStateError || s == ShardStateCancelled
}

func (s ShardStateType) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *ShardStateType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	for typ := ShardStateNew; typ <= ShardStateCompleted; typ++ {
		if equal(typ.String(), name) {
			*s = typ
			return
		}
	}
	return
}

// ShardID represents a unique identifier for a shard across all jobs.
type ShardID struct {
	JobID string `json:"JobID,omitempty"`
	Index int    `json:"Index,omitempty"`
}

func (shard ShardID) ID() string {
	return fmt.Sprintf("%s:%d", shard.JobID, shard.Index)
}

func (shard ShardID) String() string {
	return shard.ID()
}

// ShardState represents the state of a shard in a job that represents an aggregate view across
// the nodes that are executing the shard.
type ShardState struct {
	// JobID is the unique identifier for the job
	JobID string `json:"JobID"`
	// ShardIndex is the index of the shard in the job
	ShardIndex int `json:"ShardIndex"`
	// Executions is a list of executions of the shard across the nodes.
	// A new execution is created when a node is selected to execute the shard, and a node can have multiple executions for the same
	// shard due to retries, but there can only be a single active execution per node at any given time.
	Executions []ExecutionState `json:"Executions"`
	// State is the current state of the shard
	State ShardStateType `json:"State"`
	// Version is the version of the shard state. It is incremented every time the shard state is updated.
	Version int `json:"Version"`
	// CreateTime is the time when the shard was created, which is the same as the job creation time.
	CreateTime time.Time `json:"CreateTime"`
	// UpdateTime is the time when the shard state was last updated.
	UpdateTime time.Time `json:"UpdateTime"`
}

// ID returns the shard ID for this execution
func (s ShardState) ID() ShardID {
	return ShardID{JobID: s.JobID, Index: s.ShardIndex}
}
