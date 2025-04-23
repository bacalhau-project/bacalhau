package analytics

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Execution event type constants
const (
	// TerminalExecutionEventType is the event type for an execution that has reached a terminal state
	TerminalExecutionEventType = "bacalhau.execution_v1.terminal"

	// CreatedExecutionEventType is the event type for an execution that has been created
	CreatedExecutionEventType = "bacalhau.execution_v1.create"

	// ComputeMessageExecutionEventType is the event type for compute node messages related to an execution
	ComputeMessageExecutionEventType = "bacalhau.execution_v1.compute_message"
)

// NewCreatedExecutionEvent creates a new event for when an execution is created.
//
// Parameters:
//   - e: The execution that was created
//
// Returns an Event representing the execution creation.
func NewCreatedExecutionEvent(e models.Execution) Event {
	return createExecutionEvent(e, CreatedExecutionEventType)
}

// NewTerminalExecutionEvent creates a new event for when an execution reaches a terminal state.
//
// Parameters:
//   - e: The execution that reached a terminal state
//
// Returns an Event representing the execution terminal state.
func NewTerminalExecutionEvent(e models.Execution) Event {
	return createExecutionEvent(e, TerminalExecutionEventType)
}

// createExecutionEvent creates a new execution event with the given type.
// This function handles the common logic for both created and terminal execution events.
//
// Parameters:
//   - e: The execution to create an event for
//   - eventType: The type of event to create (created or terminal)
//
// Returns an Event with all execution properties populated.
func createExecutionEvent(e models.Execution, eventType string) Event {
	// Process allocated resources for each task
	resources := extractResourcesFromExecution(e)

	// Extract run output details if available
	stdoutTruncated, stderrTruncated, exitCode := extractRunOutputDetails(e)

	// Extract error codes from state details
	desiredStateErrorCode, computeStateErrorCode := extractStateErrorCodes(e)

	// Build the complete properties map
	props := EventProperties{
		// ID fields
		"job_id":        e.JobID,
		"execution_id":  e.ID,
		"evaluation_id": e.EvalID,

		// Name fields
		"name_set":       e.Name == "",
		"node_name_hash": hashString(e.NodeID),
		"namespace_hash": hashString(e.Namespace),

		// Resources of tasks in execution
		"resources": resources,

		// States
		"desired_state":            e.DesiredState.StateType.String(),
		"desired_state_error_code": desiredStateErrorCode,
		"compute_state":            e.ComputeState.StateType.String(),
		"compute_state_error_code": computeStateErrorCode,

		// Publisher if any
		"publisher_type": e.PublishedResult.Type,

		// Run results if any
		"run_result_stdout_truncated": stdoutTruncated,
		"run_result_stderr_truncated": stderrTruncated,
		"run_result_exit_code":        exitCode,

		// IDs of related models
		"previous_execution": e.PreviousExecution,
		"next_execution":     e.NextExecution,
		"followup_eval_id":   e.FollowupEvalID,

		// Versioning and time
		"revision":    e.Revision,
		"create_time": time.Unix(0, e.CreateTime).UTC(),
		"modify_time": time.Unix(0, e.ModifyTime).UTC(),
	}

	return NewEvent(eventType, props)
}

// NewComputeMessageExecutionEvent creates a new event for compute node messages.
// This event contains minimal information focused on the compute message.
//
// Parameters:
//   - e: The execution that the compute message is related to
//
// Returns an Event with compute message information.
func NewComputeMessageExecutionEvent(e models.Execution) Event {
	var errorCode string
	if e.ComputeState.Details != nil {
		errorCode = e.ComputeState.Details[models.DetailsKeyErrorCode]
	}

	props := EventProperties{
		"job_id":                   e.JobID,
		"execution_id":             e.ID,
		"compute_message":          e.ComputeState.Message,
		"compute_state_error_code": errorCode,
	}

	return NewEvent(ComputeMessageExecutionEventType, props)
}

// extractResourcesFromExecution processes an execution's allocated resources
// and returns a map of task resources with hashed task names for privacy.
//
// Parameters:
//   - e: The execution to extract resources from
//
// Returns a map of hashed task names to resource information.
func extractResourcesFromExecution(e models.Execution) map[string]resource {
	if e.AllocatedResources == nil {
		return nil
	}
	resources := make(map[string]resource, len(e.AllocatedResources.Tasks))

	for taskName, taskResources := range e.AllocatedResources.Tasks {
		gpuTypes := make([]gpuInfo, len(taskResources.GPUs))
		for i, gpu := range taskResources.GPUs {
			gpuTypes[i] = gpuInfo{
				Name:   gpu.Name,
				Vendor: string(gpu.Vendor),
			}
		}

		// Hash the task name for privacy
		hashedTaskName := hashString(taskName)
		resources[hashedTaskName] = resource{
			CPUUnits:    taskResources.CPU,
			MemoryBytes: taskResources.Memory,
			DiskBytes:   taskResources.Disk,
			GPUCount:    taskResources.GPU,
			GPUTypes:    gpuTypes,
		}
	}

	return resources
}

// extractRunOutputDetails extracts the run output details from an execution.
// It handles the case where RunOutput might be nil.
//
// Parameters:
//   - e: The execution to extract run output details from
//
// Returns stdout/stderr truncation flags and exit code.
func extractRunOutputDetails(e models.Execution) (stdoutTruncated, stderrTruncated bool, exitCode int) {
	if e.RunOutput != nil {
		stdoutTruncated = e.RunOutput.StdoutTruncated
		stderrTruncated = e.RunOutput.StderrTruncated
		exitCode = e.RunOutput.ExitCode
	}
	return
}

// extractStateErrorCodes extracts error codes from the desired and compute states.
// It handles the case where Details might be nil.
//
// Parameters:
//   - e: The execution to extract state error codes from
//
// Returns desired state and compute state error codes.
func extractStateErrorCodes(e models.Execution) (desiredStateErrorCode, computeStateErrorCode string) {
	if e.DesiredState.Details != nil {
		desiredStateErrorCode = e.DesiredState.Details[models.DetailsKeyErrorCode]
	}
	if e.ComputeState.Details != nil {
		computeStateErrorCode = e.ComputeState.Details[models.DetailsKeyErrorCode]
	}
	return
}
