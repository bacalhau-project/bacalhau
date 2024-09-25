package analytics

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const ExecutionEventType = "bacalhau.execution_v1"

type ExecutionEvent struct {
	ExecutionID string `json:"execution_id,omitempty"`
	JobID       string `json:"job_id,omitempty"`
	EvalID      string `json:"eval_id,omitempty"`

	// name of execution
	Name string `json:"name,omitempty"`
	// hash of the node name running execution
	NodeNameHash string `json:"node_name_hash,omitempty"`
	// hash of namespace execution is running in
	NamespaceHash string `json:"namespace_hash,omitempty"`

	// AllocatedResources
	// NB: total execution resources for all tasks in execution
	TotalCPUUnits    float64 `json:"total_cpu_units,omitempty"`
	TotalMemoryBytes uint64  `json:"total_memory_bytes,omitempty"`
	TotalDiskBytes   uint64  `json:"total_disk_bytes,omitempty"`
	TotalGPUCount    uint64  `json:"total_gpu_count,omitempty"`

	DesiredState        string `json:"desired_state,omitempty"`
	DesiredStateMessage string `json:"desired_state_message,omitempty"`
	ComputeState        string `json:"compute_state,omitempty"`
	ComputeStateMessage string `json:"compute_state_message,omitempty"`

	PublishedResultType string `json:"published_result_type,omitempty"`

	//RunCommandResult
	RunResultStdoutTruncated bool `json:"run_result_stdout_truncated,omitempty"`
	RunResultStderrTruncated bool `json:"run_result_stderr_truncated,omitempty"`
	RunResultExitCode        int  `json:"run_result_exit_code,omitempty"`
	// TODO determine if this contains PII.
	RunResultErrorMessage string `json:"run_result_error_message,omitempty"`

	PreviousExecution string `json:"previous_execution,omitempty"`
	NextExecution     string `json:"next_execution,omitempty"`
	FollowupEvalID    string `json:"followup_eval_id,omitempty"`
	Revision          uint64 `json:"revision,omitempty"`
	CreateTime        int64  `json:"create_time,omitempty"`
	ModifyTime        int64  `json:"modify_time,omitempty"`
}

func NewExecutionEvent(e models.Execution) *Event {
	totalResources := e.AllocatedResources.Total()

	var (
		stdoutTruncated bool
		stderrTruncated bool
		exitCode        int
		errorMessage    string
	)
	if e.RunOutput != nil {
		stdoutTruncated = e.RunOutput.StdoutTruncated
		stderrTruncated = e.RunOutput.StderrTruncated
		exitCode = e.RunOutput.ExitCode
		errorMessage = e.RunOutput.ErrorMsg
	}

	event := ExecutionEvent{
		ExecutionID:              e.ID,
		JobID:                    e.JobID,
		EvalID:                   e.EvalID,
		Name:                     e.Name,
		NodeNameHash:             hashString(e.NodeID),
		NamespaceHash:            hashString(e.Namespace),
		TotalCPUUnits:            totalResources.CPU,
		TotalMemoryBytes:         totalResources.Memory,
		TotalDiskBytes:           totalResources.Disk,
		TotalGPUCount:            totalResources.GPU,
		DesiredState:             e.DesiredState.StateType.String(),
		DesiredStateMessage:      e.DesiredState.Message,
		ComputeState:             e.ComputeState.StateType.String(),
		ComputeStateMessage:      e.ComputeState.Message,
		PublishedResultType:      e.PublishedResult.Type,
		RunResultStdoutTruncated: stdoutTruncated,
		RunResultStderrTruncated: stderrTruncated,
		RunResultExitCode:        exitCode,
		RunResultErrorMessage:    errorMessage,
		PreviousExecution:        e.PreviousExecution,
		NextExecution:            e.NextExecution,
		FollowupEvalID:           e.FollowupEvalID,
		Revision:                 e.Revision,
		CreateTime:               e.CreateTime,
		ModifyTime:               e.ModifyTime,
	}

	return NewEvent(ExecutionEventType, event)
}
