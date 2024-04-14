package compute

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	EventTopicExecutionBidding     models.EventTopic = "Requesting Node"
	EventTopicExecutionDownloading models.EventTopic = "Downloading Inputs"
	EventTopicExecutionPreparing   models.EventTopic = "Preparing Environment"
	EventTopicExecutionRunning     models.EventTopic = "Running Execution"
	EventTopicExecutionPublishing  models.EventTopic = "Publishing Results"
)

func RespondedToBidEvent(response *bidstrategy.BidStrategyResponse) models.Event {
	message := response.Reason
	if message == "" && response.ShouldBid {
		message = "Accepted job"
	}

	return models.Event{
		Message:   message,
		Topic:     EventTopicExecutionBidding,
		Timestamp: time.Now(),
		Details:   map[string]string{},
	}
}
