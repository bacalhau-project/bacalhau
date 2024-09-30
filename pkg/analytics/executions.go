package analytics

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const TerminalExecutionEventType = "bacalhau.execution_v1.terminal"
const CreatedExecutionEventType = "bacalhau.execution_v1.create"

type ExecutionEvent struct {
	JobID       string `json:"job_id,omitempty"`
	ExecutionID string `json:"execution_id,omitempty"`
	EvalID      string `json:"evaluation_id,omitempty"`

	NameSet       bool   `json:"name_set,omitempty"`
	NodeNameHash  string `json:"node_name_hash,omitempty"`
	NamespaceHash string `json:"namespace_hash,omitempty"`

	Resources map[string]Resource `json:"resources,omitempty"`

	DesiredState          string `json:"desired_state,omitempty"`
	DesiredStateErrorCode string `json:"desired_state_error_code,omitempty"`

	ComputeState          string `json:"compute_state,omitempty"`
	ComputeStateErrorCode string `json:"compute_state_error_code,omitempty"`

	PublishedResultType string `json:"publisher_type,omitempty"`

	RunResultStdoutTruncated bool `json:"run_result_stdout_truncated,omitempty"`
	RunResultStderrTruncated bool `json:"run_result_stderr_truncated,omitempty"`
	RunResultExitCode        int  `json:"run_result_exit_code,omitempty"`

	PreviousExecution string `json:"previous_execution,omitempty"`
	NextExecution     string `json:"next_execution,omitempty"`
	FollowupEvalID    string `json:"followup_eval_id,omitempty"`

	Revision   uint64    `json:"revision,omitempty"`
	CreateTime time.Time `json:"create_time,omitempty"`
	ModifyTime time.Time `json:"modify_time,omitempty"`
}

func NewCreatedExecutionEvent(e models.Execution) *Event {
	return NewEvent(CreatedExecutionEventType, newExecutionEvent(e))
}

func NewTerminalExecutionEvent(e models.Execution) *Event {
	return NewEvent(TerminalExecutionEventType, newExecutionEvent(e))
}

func newExecutionEvent(e models.Execution) ExecutionEvent {
	resources := make(map[string]Resource, len(e.AllocatedResources.Tasks))
	for taskName, taskResources := range e.AllocatedResources.Tasks {
		gpuTypes := make([]GPUInfo, len(taskResources.GPUs))
		for i, gpu := range taskResources.GPUs {
			gpuTypes[i] = GPUInfo{
				Name:   gpu.Name,
				Vendor: string(gpu.Vendor),
			}
		}
		// we hash the taskName here for privacy
		resources[hashString(taskName)] = Resource{
			CPUUnits:    taskResources.CPU,
			MemoryBytes: taskResources.Memory,
			DiskBytes:   taskResources.Disk,
			GPUCount:    taskResources.GPU,
			GPUTypes:    gpuTypes,
		}
	}

	var (
		stdoutTruncated bool
		stderrTruncated bool
		exitCode        int
	)
	if e.RunOutput != nil {
		stdoutTruncated = e.RunOutput.StdoutTruncated
		stderrTruncated = e.RunOutput.StderrTruncated
		exitCode = e.RunOutput.ExitCode
	}

	var (
		desiredStateErrorCode string
		computeStateErrorCode string
	)

	if e.DesiredState.Details != nil {
		desiredStateErrorCode = e.DesiredState.Details[models.DetailsKeyErrorCode]
	}
	if e.ComputeState.Details != nil {
		computeStateErrorCode = e.ComputeState.Details[models.DetailsKeyErrorCode]
	}

	return ExecutionEvent{
		// ID fields.
		JobID:       e.JobID,
		ExecutionID: e.ID,
		EvalID:      e.EvalID,

		// name fields.
		NameSet:       e.Name == "",
		NodeNameHash:  hashString(e.NodeID),
		NamespaceHash: hashString(e.Namespace),

		// resources of tasks in execution.
		// NB: currently this isn't populated when creating executions.
		Resources: resources,

		// states.
		DesiredState:          e.DesiredState.StateType.String(),
		DesiredStateErrorCode: desiredStateErrorCode,
		ComputeState:          e.ComputeState.StateType.String(),
		ComputeStateErrorCode: computeStateErrorCode,

		// publisher if any.
		PublishedResultType: e.PublishedResult.Type,

		// run results if any.
		RunResultStdoutTruncated: stdoutTruncated,
		RunResultStderrTruncated: stderrTruncated,
		RunResultExitCode:        exitCode,

		// IDs of related models.
		PreviousExecution: e.PreviousExecution,
		NextExecution:     e.NextExecution,
		FollowupEvalID:    e.FollowupEvalID,

		// versioning and time.
		Revision:   e.Revision,
		CreateTime: time.Unix(0, e.CreateTime).UTC(),
		ModifyTime: time.Unix(0, e.ModifyTime).UTC(),
	}
}
