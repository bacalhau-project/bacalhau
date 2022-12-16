package requester

import (
	"context"
	"fmt"
	"time"

	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
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
	JobStore                   localdb.LocalDB
	Scheduler                  *Scheduler
	Verifiers                  verifier.VerifierProvider
	StorageProviders           storage.StorageProvider
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration
}

// BaseEndpoint base implementation of requester Endpoint
type BaseEndpoint struct {
	id                         string
	publicKey                  []byte
	jobStore                   localdb.LocalDB
	scheduler                  *Scheduler
	verifiers                  verifier.VerifierProvider
	storageProviders           storage.StorageProvider
	minJobExecutionTimeout     time.Duration
	defaultJobExecutionTimeout time.Duration
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	return &BaseEndpoint{
		id:                         params.ID,
		publicKey:                  params.PublicKey,
		jobStore:                   params.JobStore,
		scheduler:                  params.Scheduler,
		verifiers:                  params.Verifiers,
		storageProviders:           params.StorageProviders,
		minJobExecutionTimeout:     params.MinJobExecutionTimeout,
		defaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,
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
	jobCtx, span := node.newRootSpanForJob(ctx, jobID)
	defer span.End()

	// TODO: Should replace the span above, with the below, but I don't understand how/why we're tracing contexts in a variable.
	// Specifically tracking them all in ctrl.jobContexts
	// ctx, span := system.NewRootSpan(ctx, system.GetTracer(), "pkg/controller.SubmitJob")
	// defer span.End()

	executionPlan, err := jobutils.GenerateExecutionPlan(ctx, *data.Spec, node.storageProviders)
	if err != nil {
		return &model.Job{}, fmt.Errorf("error generating execution plan: %s", err)
	}

	job := &model.Job{
		APIVersion: data.APIVersion,
		Metadata: model.Metadata{
			ID:        jobID,
			ClientID:  data.ClientID,
			CreatedAt: time.Now(),
		},
		Status: model.JobStatus{
			Requester: model.JobRequester{
				RequesterNodeID:    node.id,
				RequesterPublicKey: node.publicKey,
			},
		},
		Spec: *data.Spec,
	}
	job.Spec.Deal = data.Spec.Deal
	job.Spec.ExecutionPlan = executionPlan

	// set a default timeout value if one is not passed or below an acceptable value
	if job.Spec.GetTimeout() <= node.minJobExecutionTimeout {
		job.Spec.Timeout = node.defaultJobExecutionTimeout.Seconds()
	}

	err = node.scheduler.StartJob(jobCtx, StartJobRequest{
		Job: *job,
	})
	if err != nil {
		return &model.Job{}, fmt.Errorf("error starting job: %w", err)
	}

	return job, nil
}
func (node *BaseEndpoint) UpdateDeal(ctx context.Context, jobID string, deal model.Deal) error {
	//TODO: Is there an action to take here?
	return node.jobStore.UpdateJobDeal(ctx, jobID, deal)
}

func (node *BaseEndpoint) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	//TODO implement me
	panic("implement me")
}

func (node *BaseEndpoint) newRootSpanForJob(ctx context.Context, jobID string) (context.Context, trace.Span) {
	return system.Span(ctx, "requester", "JobLifecycle",
		// job lifecycle spans go in their own, dedicated trace
		trace.WithNewRoot(),

		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, node.id),
			attribute.String(model.TracerAttributeNameJobID, jobID),
		),
	)
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
