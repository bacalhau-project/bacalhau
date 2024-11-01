package compute

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	EventTopicExecution            models.EventTopic = "Execution"
	EventTopicExecutionScanning    models.EventTopic = "Exec Scanning"
	EventTopicExecutionDownloading models.EventTopic = "Downloading Inputs"
	EventTopicExecutionPreparing   models.EventTopic = "Preparing Environment"
	EventTopicExecutionRunning     models.EventTopic = "Running Execution"
	EventTopicExecutionPublishing  models.EventTopic = "Publishing Results"
	EventTopicRestart              models.EventTopic = "Restart"
)

const (
	execCompletedMessage        = "Completed successfully"
	execRunningMessage          = "Running"
	execFailingDueToNodeRestart = "Failing due to node restart"
)

func ExecCompletedEvent() *models.Event {
	return models.NewEvent(EventTopicExecution).WithMessage(execCompletedMessage)
}

func ExecRunningEvent() *models.Event {
	return models.NewEvent(EventTopicExecution).WithMessage(execRunningMessage)
}

// ExecFailedDueToNodeRestartEvent returns an event indicating that the execution failed due to a node restart
func ExecFailedDueToNodeRestartEvent() *models.Event {
	return models.NewEvent(EventTopicExecution).WithMessage(execFailingDueToNodeRestart).WithFailsExecution(true)
}
