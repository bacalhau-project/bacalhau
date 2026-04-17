package orchestrator

import (
	"fmt"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	EventTopicJobSubmission    models.EventTopic = "Submission"
	EventTopicJobStateUpdate   models.EventTopic = "State Update"
	EventTopicJobScheduling    models.EventTopic = "Scheduling"
	EventTopicJobQueueing      models.EventTopic = "Queueing"
	EventTopicExecutionTimeout models.EventTopic = "Exec Timeout"
	EventTopicJobTimeout       models.EventTopic = "Job Timeout"
	EventTopicExecution        models.EventTopic = "Execution"
)

const (
	jobSubmittedMessage        = "Job submitted"
	jobUpdatedMessage          = "Job updated"
	jobTranslatedMessage       = "Job tasks translated to new type"
	jobQueuedMessage           = "Job queued"
	jobStopRequestedMessage    = "Job requested to stop before completion"
	jobRerunRequestedMessage   = "Job rerun requested"
	jobExhaustedRetriesMessage = "Job failed because it has been retried too many times"
	JobTimeoutMessage          = "Job timed out"
	jobExecutionsFailedMessage = "Job failed because one or more executions failed"

	execCompletedMessage                 = "Completed successfully"
	execRunningMessage                   = "Running"
	execStoppedByJobStopMessage          = "Execution stop requested because job has been stopped"
	execStoppedByNodeUnhealthyMessage    = "Execution stop requested because node has disappeared"
	execStoppedByNodeRejectedMessage     = "Execution stop requested because node has been rejected"
	execStoppedByOversubscriptionMessage = "Execution stop requested because there are more executions than needed"
	execStoppedDueToJobFailureMessage    = "Execution stopped due to job failure"
	execStoppedForJobUpdateMessage       = "Execution stopped for job update"

	executionTimeoutMessage = "Execution timed out"

	// TODO: message is duplicated in compute/errors.go. Find a better place for common errors
	timeoutHint = "Increase the task timeout or allocate more resources"
)

func event(topic models.EventTopic, msg string, details map[string]string) models.Event {
	return models.Event{
		Message:   msg,
		Topic:     topic,
		Timestamp: time.Now(),
		Details:   details,
	}
}

func JobSubmittedEvent() models.Event {
	return event(EventTopicJobSubmission, jobSubmittedMessage, map[string]string{})
}

func JobUpdatedEvent() models.Event {
	return event(EventTopicJobSubmission, jobUpdatedMessage, map[string]string{})
}

func JobTranslatedEvent(old, new *models.Job) models.Event {
	return event(EventTopicJobSubmission, jobTranslatedMessage, map[string]string{
		"PreviousTaskType": old.Task().Engine.Type,
		"NewTaskType":      new.Task().Engine.Type,
	})
}

func JobStateUpdateEvent(new models.JobStateType, message ...string) models.Event {
	eventMessage := new.String()
	if len(message) > 0 && message[0] != "" {
		eventMessage += fmt.Sprintf(". %s", strings.Join(message, ". "))
	}
	return *models.NewEvent(EventTopicJobStateUpdate).
		WithMessage(eventMessage).
		WithDetail(models.DetailsKeyNewState, new.String())
}

func JobStoppedEvent(reason string) models.Event {
	return event(EventTopicJobScheduling, jobStopRequestedMessage, map[string]string{
		"Reason": reason,
	})
}

func JobExhaustedRetriesEvent() models.Event {
	return event(EventTopicJobScheduling, jobExhaustedRetriesMessage, map[string]string{})
}

func JobTimeoutEvent(timeout time.Duration) models.Event {
	e := models.NewEvent(EventTopicJobTimeout).
		WithError(fmt.Errorf("%s. Job took longer than %s", JobTimeoutMessage, timeout)).
		WithHint(timeoutHint).
		WithFailsExecution(true)
	return *e
}

func JobExecutionsFailedEvent() models.Event {
	return event(EventTopicJobScheduling, jobExecutionsFailedMessage, map[string]string{})
}

func JobQueueingEvent(reason string) models.Event {
	message := jobQueuedMessage
	if reason != "" {
		message = fmt.Sprintf("%s. %s", message, reason)
	}
	return *models.NewEvent(EventTopicJobQueueing).WithMessage(message)
}

func JobRerunEvent(reason string) models.Event {
	return event(EventTopicJobSubmission, jobRerunRequestedMessage, map[string]string{
		"Reason": reason,
	})
}

func ExecCreatedEvent(execution *models.Execution) models.Event {
	return *models.NewEvent(EventTopicJobScheduling).
		WithMessage(fmt.Sprintf("Requested execution on %s", idgen.ShortNodeID(execution.NodeID))).
		WithDetail("NodeID", execution.NodeID)
}

func ExecCompletedEvent() models.Event {
	return *models.NewEvent(EventTopicExecution).WithMessage(execCompletedMessage)
}

func ExecRunningEvent() models.Event {
	return *models.NewEvent(EventTopicExecution).WithMessage(execRunningMessage)
}

func ExecStoppedByJobStopEvent() models.Event {
	return event(EventTopicJobScheduling, execStoppedByJobStopMessage, map[string]string{})
}

func ExecStoppedByNodeUnhealthyEvent() models.Event {
	return event(EventTopicJobScheduling, execStoppedByNodeUnhealthyMessage, map[string]string{})
}

func ExecStoppedByExecutionTimeoutEvent(timeout time.Duration) models.Event {
	e := models.NewEvent(EventTopicExecutionTimeout).
		WithError(fmt.Errorf("%s. Execution took longer than %s", executionTimeoutMessage, timeout)).
		WithHint(timeoutHint).
		WithFailsExecution(true)
	return *e
}

func ExecStoppedByNodeRejectedEvent() models.Event {
	return event(EventTopicJobScheduling, execStoppedByNodeRejectedMessage, map[string]string{})
}

func ExecStoppedByOversubscriptionEvent() models.Event {
	return event(EventTopicJobScheduling, execStoppedByOversubscriptionMessage, map[string]string{})
}

func ExecStoppedDueToJobFailureEvent() models.Event {
	return *models.NewEvent(EventTopicJobScheduling).WithMessage(execStoppedDueToJobFailureMessage)
}

func ExecStoppedForJobUpdateEvent() models.Event {
	return event(EventTopicJobScheduling, execStoppedForJobUpdateMessage, map[string]string{})
}
