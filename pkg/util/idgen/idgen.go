package idgen

import "github.com/google/uuid"

const (
	// JobIDPrefix is the prefix of job ID.
	JobIDPrefix = "j-"

	// ExecutionIDPrefix is the prefix of execution ID.
	ExecutionIDPrefix = "e-"

	// EvaluationIDPrefix is the prefix of evaluation ID.
	EvaluationIDPrefix = "v-"

	// NodeIDPrefix is the prefix of node ID.
	NodeIDPrefix = "n-"

	// TaskNamePrefix is the prefix of a system generated task name.
	TaskNamePrefix = "t-name-"

	// JobNamePrefix is the prefix of a system generated job name.
	JobNamePrefix = "j-name-"
)

// newWithPrefix generates a new UUID with the given prefix.
func newWithPrefix(prefix string) string {
	return prefix + uuid.NewString()
}

// NewJobID generates a new job ID.
func NewJobID() string {
	return newWithPrefix(JobIDPrefix)
}

// NewExecutionID generates a new execution ID.
func NewExecutionID() string {
	return newWithPrefix(ExecutionIDPrefix)
}

// NewEvaluationID generates a new evaluation ID.
func NewEvaluationID() string {
	return newWithPrefix(EvaluationIDPrefix)
}

func NewTaskName() string {
	return newWithPrefix(TaskNamePrefix)
}

func NewJobName() string {
	return newWithPrefix(JobNamePrefix)
}
