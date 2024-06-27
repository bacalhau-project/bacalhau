package orchestrator

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	EventTopicJobSubmission    models.EventTopic = "Submission"
	EventTopicJobScheduling    models.EventTopic = "Scheduling"
	EventTopicJobQueueing      models.EventTopic = "Queueing"
	EventTopicExecutionTimeout models.EventTopic = "Exec Timeout"
	EventTopicJobTimeout       models.EventTopic = "Job Timeout"
)

const (
	jobSubmittedMessage        = "Job submitted"
	jobTranslatedMessage       = "Job tasks translated to new type"
	jobQueuedMessage           = "Job queued"
	jobStopRequestedMessage    = "Job requested to stop before completion"
	jobExhaustedRetriesMessage = "Job failed because it has been retried too many times"
	JobTimeoutMessage          = "Job timed out"

	execStoppedByJobStopMessage          = "Execution stop requested because job has been stopped"
	execStoppedByNodeUnhealthyMessage    = "Execution stop requested because node has disappeared"
	execStoppedByNodeRejectedMessage     = "Execution stop requested because node has been rejected"
	execStoppedByOversubscriptionMessage = "Execution stop requested because there are more executions than needed"
	execRejectedByNodeMessage            = "Node responded to execution run request"
	execFailedMessage                    = "Execution did not complete successfully"

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

func JobTranslatedEvent(old, new *models.Job) models.Event {
	return event(EventTopicJobSubmission, jobTranslatedMessage, map[string]string{
		"PreviousTaskType": old.Task().Engine.Type,
		"NewTaskType":      new.Task().Engine.Type,
	})
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

func JobQueueingEvent(reason string) models.Event {
	message := jobQueuedMessage
	if reason != "" {
		message = fmt.Sprintf("%s. %s", message, reason)
	}
	return *models.NewEvent(EventTopicJobQueueing).WithMessage(message)
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
