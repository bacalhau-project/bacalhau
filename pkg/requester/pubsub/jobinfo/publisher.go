package jobinfo

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
)

type Envelope struct {
	// Top level fields for job info decoding
	APIVersion string
	ID         string
	// Actual payload
	Info model.JobWithInfo
}

type PublisherParams struct {
	JobStore jobstore.Store
	PubSub   pubsub.PubSub[Envelope]
}

type Publisher struct {
	jobStore jobstore.Store
	pubSub   pubsub.PubSub[Envelope]
}

func NewPublisher(params PublisherParams) *Publisher {
	return &Publisher{
		jobStore: params.JobStore,
		pubSub:   params.PubSub,
	}
}

func (p Publisher) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	if event.EventName.IsTerminal() {
		job, err := p.jobStore.GetJob(ctx, event.JobID)
		if err != nil {
			return err
		}

		jobState, err := p.jobStore.GetJobState(ctx, event.JobID)
		if err != nil {
			return err
		}

		history, err := p.jobStore.GetJobHistory(ctx, event.JobID, jobstore.JobHistoryFilterOptions{})
		if err != nil {
			return err
		}

		envelope := Envelope{
			APIVersion: job.APIVersion,
			ID:         job.ID(),
			Info: model.JobWithInfo{
				Job:     job,
				State:   jobState,
				History: history,
			},
		}

		return p.pubSub.Publish(ctx, envelope)
	}
	return nil
}
