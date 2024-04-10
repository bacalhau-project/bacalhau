package orchestrator

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	jobSubmittedMessage        = "Job submitted"
	jobTranslatedMessage       = "Job tasks translated to new type"
	jobStopRequestedMessage    = "Job requested to stop before completion"
	jobExhaustedRetriesMessage = "Job failed because it has been retried too many times"
	jobNotEnoughNodesMessage   = "Job failed because not enough nodes are suitable to run it"

	execStoppedByJobStopMessage          = "Execution stop requested because job has been stopped"
	execStoppedByNodeUnhealthyMessage    = "Execution stop requested because node has disappeared"
	execStoppedByNodeRejectedMessage     = "Execution stop requested because node has been rejected"
	execStoppedByOversubscriptionMessage = "Execution stop requested because there are more executions than needed"
	execRejectedByNodeMessage            = "Node responded to execution run request"
	execFailedMessage                    = "Execution did not complete successfully"
)

func event(msg string, details map[string]string) models.Event {
	return models.Event{
		Message:   msg,
		Timestamp: time.Now(),
		Details:   details,
	}
}

func JobSubmittedEvent() models.Event {
	return event(jobSubmittedMessage, map[string]string{})
}

func JobTranslatedEvent(old, new *models.Job) models.Event {
	return event(jobTranslatedMessage, map[string]string{
		"PreviousTaskType": old.Task().Engine.Type,
		"NewTaskType":      new.Task().Engine.Type,
	})
}

func JobStoppedEvent(reason string) models.Event {
	return event(jobStopRequestedMessage, map[string]string{
		"Reason": reason,
	})
}

func JobExhaustedRetriesEvent() models.Event {
	return event(jobExhaustedRetriesMessage, map[string]string{})
}

func JobNotEnoughNodesEvent() models.Event {
	return event(jobNotEnoughNodesMessage, map[string]string{})
}

func ExecStoppedByJobStopEvent() models.Event {
	return event(execStoppedByJobStopMessage, map[string]string{})
}

func ExecStoppedByNodeUnhealthyEvent() models.Event {
	return event(execStoppedByNodeUnhealthyMessage, map[string]string{})
}

func ExecStoppedByNodeRejectedEvent() models.Event {
	return event(execStoppedByNodeRejectedMessage, map[string]string{})
}

func ExecStoppedByOversubscriptionEvent() models.Event {
	return event(execStoppedByOversubscriptionMessage, map[string]string{})
}

func BidResponseFromNodeEvent(reason string) models.Event {
	return event(execRejectedByNodeMessage, map[string]string{
		"Reason": reason,
	})
}

func ExecutionFailedEvent(reason string) models.Event {
	return event(execFailedMessage, map[string]string{
		"Error": reason,
	})
}
