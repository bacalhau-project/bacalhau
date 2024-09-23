package compute

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	EventTopicExecutionCompatibilityCheck models.EventTopic = "Compatibility Check"
	EventTopicExecutionDownloading        models.EventTopic = "Downloading Inputs"
	EventTopicExecutionPreparing          models.EventTopic = "Preparing Environment"
	EventTopicExecutionRunning            models.EventTopic = "Running Execution"
	EventTopicExecutionPublishing         models.EventTopic = "Publishing Results"
)

func RespondedToBidEvent(response *bidstrategy.BidStrategyResponse) models.Event {
	return models.Event{
		Message:   response.Reason,
		Topic:     EventTopicExecutionCompatibilityCheck,
		Timestamp: time.Now(),
		Details:   map[string]string{},
	}
}
