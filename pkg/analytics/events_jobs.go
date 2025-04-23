package analytics

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	// SubmitJobEventType is the event type for a job that has been submitted to an orchestrator.
	SubmitJobEventType = "bacalhau.job_v1.submit"

	// TerminalJobEventType is the event type for a job that has reached a terminal state.
	TerminalJobEventType = "bacalhau.job_v1.terminal"
)

// NewSubmitJobEvent creates a new job submit event from a job.
// This event is emitted when a job is submitted to an orchestrator.
//
// Parameters:
//   - j: The job that was submitted
//   - warnings: Optional warnings that occurred during job submission
//
// Returns an Event representing the job submission.
func NewSubmitJobEvent(j models.Job, jobID string, submissionError error, warnings ...string) Event {
	props := createCommonJobProperties(j)

	// Add submit-specific properties
	if jobID != "" {
		props["job_id"] = jobID
	}
	if submissionError != nil {
		props["error"] = submissionError.Error()
	}
	if len(warnings) > 0 {
		props["warnings"] = warnings
	}

	// Return an efficient baseEvent with pre-computed properties
	return NewEvent(SubmitJobEventType, props)
}

// NewJobTerminalEvent creates a new job terminal event from a job.
// This event is emitted when a job reaches a terminal state (completed, failed, etc.).
//
// Parameters:
//   - j: The job that reached a terminal state
//
// Returns an Event representing the job terminal state.
func NewJobTerminalEvent(j models.Job) Event {
	props := createCommonJobProperties(j)

	// Add terminal-specific properties
	props["state"] = j.State.StateType.String()

	// Return an efficient baseEvent with pre-computed properties
	return NewEvent(TerminalJobEventType, props)
}

// createCommonJobProperties creates a map of properties common to all job events.
// This is a helper function to reduce duplication between event types.
//
// Parameters:
//   - j: The job to extract properties from
//
// Returns a map with common job and task properties.
func createCommonJobProperties(j models.Job) EventProperties {
	t := j.Task()
	if t == nil {
		return EventProperties{}
	}

	return EventProperties{
		"job_id":                  j.ID,
		"name_set":                j.ID != j.Name,
		"namespace_hash":          hashString(j.Namespace),
		"type":                    j.Type,
		"count":                   j.Count,
		"labels_count":            len(j.Labels),
		"meta_count":              len(j.Meta),
		"version":                 j.Version,
		"revision":                j.Revision,
		"create_time":             time.Unix(0, j.CreateTime).UTC(),
		"modify_time":             time.Unix(0, j.ModifyTime).UTC(),
		"task_name_hash":          hashString(t.Name),
		"task_engine_type":        t.Engine.Type,
		"task_publisher_type":     t.Publisher.Type,
		"task_env_var_count":      len(t.Env),
		"task_meta_count":         len(t.Meta),
		"task_input_source_types": getInputSourceTypes(t),
		"task_result_path_count":  len(t.ResultPaths),
		"task_docker_image":       getDockerImageTelemetry(t.Engine),
		"resources":               newResourceFromConfig(t.ResourcesConfig),
		"task_network_type":       t.Network.Type.String(),
		"task_domains_count":      len(t.Network.Domains),
		"task_execution_timeout":  t.Timeouts.ExecutionTimeout,
		"task_queue_timeout":      t.Timeouts.QueueTimeout,
		"task_total_timeout":      t.Timeouts.TotalTimeout,
	}
}
