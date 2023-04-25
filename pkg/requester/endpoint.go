package requester

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/jobtransform"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
)

type BaseEndpointParams struct {
	ID                         string
	PublicKey                  []byte
	Queue                      Queue
	Selector                   bidstrategy.SemanticBidStrategy
	Store                      jobstore.Store
	ComputeEndpoint            compute.Endpoint
	Verifiers                  verifier.VerifierProvider
	StorageProviders           storage.StorageProvider
	MinJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration
	GetBiddingCallback         func() *url.URL
}

// BaseEndpoint base implementation of requester Endpoint
type BaseEndpoint struct {
	id         string
	queue      Queue
	store      jobstore.Store
	computesvc compute.Endpoint
	selector   bidstrategy.SemanticBidStrategy
	callback   func() *url.URL
	transforms []jobtransform.Transformer
}

func NewBaseEndpoint(params *BaseEndpointParams) *BaseEndpoint {
	transforms := []jobtransform.Transformer{
		jobtransform.NewInlineStoragePinner(params.StorageProviders),
		jobtransform.NewTimeoutApplier(params.MinJobExecutionTimeout, params.DefaultJobExecutionTimeout),
		jobtransform.NewRequesterInfo(params.ID, params.PublicKey),
		jobtransform.RepoExistsOnIPFS(params.StorageProviders),
		jobtransform.NewPublisherMigrator(),
	}

	return &BaseEndpoint{
		id:         params.ID,
		queue:      params.Queue,
		computesvc: params.ComputeEndpoint,
		selector:   params.Selector,
		store:      params.Store,
		transforms: transforms,
		callback:   params.GetBiddingCallback,
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
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/requester.BaseEndpoint.SubmitJob",
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

	err = node.store.CreateJob(ctx, *job)
	if err != nil {
		return job, err
	}

	err = node.queue.EnqueueJob(ctx, *job)
	if err != nil {
		return job, err
	}

	selectRequest := bidstrategy.BidStrategyRequest{NodeID: node.id, Job: *job}
	if url := node.callback(); url != nil {
		selectRequest.Callback = url
	}

	response, err := node.selector.ShouldBid(ctx, selectRequest)
	if err != nil {
		return job, err
	}

	return job, node.handleBidResponse(ctx, *job, response)
}

func (node *BaseEndpoint) ApproveJob(ctx context.Context, approval bidstrategy.ModerateJobRequest) error {
	// We deliberately expect this to be the empty string if unset. This is so
	// that if this env variable is (accidentally) left unset, no jobs can be
	// approved because an empty ClientID is invalid.
	approvingClient := os.Getenv("BACALHAU_JOB_APPROVER")
	if approval.ClientID != approvingClient {
		return errors.New("approval submitted by unknown client")
	}

	job, err := node.store.GetJob(ctx, approval.JobID)
	if err != nil {
		return err
	}

	return node.handleBidResponse(ctx, job, approval.Response)
}

func (node *BaseEndpoint) CancelJob(ctx context.Context, request CancelJobRequest) (CancelJobResult, error) {
	return node.queue.CancelJob(ctx, request)
}

func (node *BaseEndpoint) ReadLogs(ctx context.Context, request ReadLogsRequest) (ReadLogsResponse, error) {
	emptyResponse := ReadLogsResponse{}

	jobstate, err := node.store.GetJobState(ctx, request.JobID)
	if err != nil {
		return emptyResponse, err
	}

	job, err := node.store.GetJob(ctx, request.JobID)
	if err != nil {
		return emptyResponse, err
	}

	nodeID := ""
	for _, e := range jobstate.Executions {
		if e.ComputeReference == request.ExecutionID {
			nodeID = e.NodeID
			break
		}
	}

	if nodeID == "" {
		return emptyResponse, fmt.Errorf("unable to find execution %s in job %s", request.ExecutionID, request.JobID)
	}

	req := compute.ExecutionLogsRequest{
		RoutingMetadata: compute.RoutingMetadata{
			SourcePeerID: job.Metadata.Requester.RequesterNodeID,
			TargetPeerID: nodeID,
		},
		ExecutionID: request.ExecutionID,
		WithHistory: request.WithHistory,
		Follow:      request.Follow,
	}

	newCtx := context.Background()
	response, err := node.computesvc.ExecutionLogs(newCtx, req)
	if err != nil {
		return emptyResponse, err
	}

	return ReadLogsResponse{Address: response.Address, ExecutionComplete: response.ExecutionFinished}, nil
}

func (node *BaseEndpoint) handleBidResponse(ctx context.Context, job model.Job, response bidstrategy.BidStrategyResponse) error {
	if response.ShouldWait {
		return node.store.UpdateJobState(ctx, jobstore.UpdateJobStateRequest{
			JobID: job.ID(),
			Condition: jobstore.UpdateJobCondition{
				ExpectedState: model.JobStateQueued,
			},
			NewState: model.JobStateQueued,
			Comment:  response.Reason,
		})
	}

	if response.ShouldBid {
		return node.queue.StartJob(ctx, StartJobRequest{Job: job})
	}

	_, err := node.queue.CancelJob(ctx, CancelJobRequest{
		JobID:         job.Metadata.ID,
		Reason:        fmt.Sprintf("job rejected: %s", response.Reason),
		UserTriggered: false,
	})
	return err
}

// Compile-time interface check:
var _ Endpoint = (*BaseEndpoint)(nil)
