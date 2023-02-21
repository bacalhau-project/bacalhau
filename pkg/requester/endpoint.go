package requester

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester/jobtransform"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BaseEndpointParams struct {
	ID                         string
	PublicKey                  []byte
	Scheduler                  *Scheduler
	Verifiers                  verifier.VerifierProvider
	StorageProviders           storage.StorageProvider
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration
}

// BaseEndpoint base implementation of requester Endpoint
type BaseEndpoint struct {
	id         string
	scheduler  *Scheduler
	transforms []jobtransform.Transformer
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	transforms := []jobtransform.Transformer{
		jobtransform.NewInlineStoragePinner(params.StorageProviders),
		jobtransform.NewTimeoutApplier(params.MinJobExecutionTimeout, params.DefaultJobExecutionTimeout),
		jobtransform.NewExecutionPlanner(params.StorageProviders),
		jobtransform.NewRequesterInfo(params.ID, params.PublicKey),
	}

	return &BaseEndpoint{
		id:         params.ID,
		scheduler:  params.Scheduler,
		transforms: transforms,
	}
}

func (node *BaseEndpoint) SubmitJob(ctx context.Context, data model.JobCreatePayload) (*model.Job, error) {
	jobUUID, err := uuid.NewRandom()
	if err != nil {
		return &model.Job{}, fmt.Errorf("error creating job id: %w", err)
	}
	jobID := jobUUID.String()

	// Creates a new root context to track a job's lifecycle for tracing. This
	// should be fine as only one node will call SubmitJob(...) - the other
	// nodes will hear about the job via events on the transport.
	jobCtx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester.BaseEndpoint.SubmitJob",
		// job lifecycle spans go in their own, dedicated trace
		trace.WithNewRoot(),
		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, node.id),
			attribute.String(model.TracerAttributeNameJobID, jobID),
		),
	)
	defer span.End()

	// TODO: Should replace the span above, with the below, but I don't understand how/why we're tracing contexts in a variable.
	// Specifically tracking them all in ctrl.jobContexts
	// ctx, span := system.NewRootSpan(ctx, system.GetTracer(), "pkg/controller.SubmitJob")
	// defer span.End()

	job := &model.Job{
		APIVersion: data.APIVersion,
		Metadata: model.Metadata{
			ID:        jobID,
			ClientID:  data.ClientID,
			CreatedAt: time.Now(),
		},
		Spec: *data.Spec,
	}

	for _, transform := range node.transforms {
		_, err = transform(ctx, job)
		if err != nil {
			return job, err
		}
	}

	err = node.scheduler.StartJob(jobCtx, StartJobRequest{
		Job: *job,
	})
	if err != nil {
		return &model.Job{}, fmt.Errorf("error starting job: %w", err)
	}

	return job, nil
}

func (node *BaseEndpoint) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	return node.scheduler.CancelJob(ctx, request)
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
