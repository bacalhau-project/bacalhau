package jobinfo

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
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

		legacyJob, err := legacy.ToLegacyJob(&job)
		if err != nil {
			return err
		}

		executions, err := p.jobStore.GetExecutions(ctx, event.JobID)
		if err != nil {
			return err
		}
		jobState, err := legacy.ToLegacyJobStatus(job, executions)
		if err != nil {
			return err
		}
		// TODO: bring back job history in the envelope
		envelope := Envelope{
			APIVersion: legacyJob.APIVersion,
			ID:         legacyJob.ID(),
			Info: model.JobWithInfo{
				Job:   *legacyJob,
				State: *jobState,
			},
		}

		return p.pubSub.Publish(ctx, envelope)
	}
	return nil
}
