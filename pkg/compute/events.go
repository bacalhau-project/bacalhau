package compute

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	EventTopicExecutionBidding     models.EventTopic = "Requesting node"
	EventTopicExecutionDownloading models.EventTopic = "Downloading inputs"
	EventTopicExecutionPreparing   models.EventTopic = "Preparing environment"
	EventTopicExecutionRunning     models.EventTopic = "Running execution"
	EventTopicExecutionPublishing  models.EventTopic = "Publishing results"
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
		Details: map[string]string{
			models.DetailsKeyIsError:        fmt.Sprint(false),
			models.DetailsKeyRetryable:      fmt.Sprint(response.ShouldWait),
			models.DetailsKeyFailsExecution: fmt.Sprint(!response.ShouldBid),
		},
	}
}
