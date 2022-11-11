package requesternode

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/google/uuid"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RequesterNode struct {
	ID                 string
	localDB            localdb.LocalDB
	localEventConsumer eventhandler.LocalEventHandler
	jobEventPublisher  eventhandler.JobEventHandler
	verifiers          verifier.VerifierProvider
	storageProviders   storage.StorageProvider
	config             RequesterNodeConfig //nolint:gocritic

	shardStateManager *shardStateMachineManager
}

func NewRequesterNode(
	ctx context.Context,
	cm *system.CleanupManager,
	nodeID string,
	localDB localdb.LocalDB,
	localEventConsumer eventhandler.LocalEventHandler,
	jobEventPublisher eventhandler.JobEventHandler,
	verifiers verifier.VerifierProvider,
	storageProviders storage.StorageProvider,
	config RequesterNodeConfig, //nolint:gocritic
) (*RequesterNode, error) {
	// TODO: instrument with trace
	useConfig := populateDefaultConfigs(config)
	requesterNode := &RequesterNode{
		ID:                 nodeID,
		localDB:            localDB,
		localEventConsumer: localEventConsumer,
		jobEventPublisher:  jobEventPublisher,
		verifiers:          verifiers,
		storageProviders:   storageProviders,
		config:             useConfig,
		shardStateManager:  newShardStateMachineManager(ctx, cm, useConfig),
	}
	return requesterNode, nil
}

func (node *RequesterNode) HandleJobEvent(ctx context.Context, event model.JobEvent) error {
	j, err := node.localDB.GetJob(ctx, event.JobID)
	if err != nil {
		return fmt.Errorf("could not get job: %s - %v", event.JobID, err)
	}

	// we only care about jobs that we own
	if j.RequesterNodeID != node.ID {
		return nil
	}

	switch event.EventName {
	case model.JobEventBid, model.JobEventResultsProposed, model.JobEventResultsPublished, model.JobEventComputeError:
		shard := model.JobShard{Job: j, Index: event.ShardIndex}
		return node.triggerStateTransition(ctx, event, shard)
	}
	return nil
}

func (node *RequesterNode) SubmitJob(ctx context.Context, data model.JobCreatePayload) (*model.Job, error) {
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

	ev := node.constructJobEvent(jobID, model.JobEventCreated)

	executionPlan, err := jobutils.GenerateExecutionPlan(ctx, data.Job.Spec, node.storageProviders)
	if err != nil {
		return &model.Job{}, fmt.Errorf("error generating execution plan: %s", err)
	}

	ev.APIVersion = data.Job.APIVersion
	ev.ClientID = data.ClientID
	ev.Spec = data.Job.Spec
	ev.Deal = data.Job.Deal
	ev.JobExecutionPlan = executionPlan

	// set a default timeout value if one is not passed or below an acceptable value
	if ev.Spec.GetTimeout() <= node.config.TimeoutConfig.MinJobExecutionTimeout {
		ev.Spec.Timeout = node.config.TimeoutConfig.DefaultJobExecutionTimeout.Seconds()
	}

	job := jobutils.ConstructJobFromEvent(ev)
	err = node.localDB.AddJob(ctx, job)
	if err != nil {
		return &model.Job{}, fmt.Errorf("error saving job id: %w", err)
	}

	node.shardStateManager.startShardsState(ctx, job, node)

	err = node.jobEventPublisher.HandleJobEvent(jobCtx, ev)
	if err != nil {
		return &model.Job{}, fmt.Errorf("error handling new job event: %s", err)
	}

	return job, nil
}

func (node *RequesterNode) UpdateDeal(ctx context.Context, jobID string, deal model.Deal) error {
	ev := node.constructJobEvent(jobID, model.JobEventDealUpdated)
	ev.Deal = deal
	return node.jobEventPublisher.HandleJobEvent(ctx, ev)
}

// Return list of active jobs in this requester node.
func (node *RequesterNode) GetActiveJobs(ctx context.Context) []ActiveJob {
	activeJobs := make([]ActiveJob, 0)

	for _, shardState := range node.shardStateManager.shardStates {
		if shardState.currentState != shardCompleted && shardState.currentState != shardError {
			activeJobs = append(activeJobs, ActiveJob{
				ShardID:             shardState.shard.ID(),
				State:               shardState.currentState.String(),
				BiddingNodesCount:   len(shardState.biddingNodes),
				CompletedNodesCount: len(shardState.completedNodes),
			})
		}
	}

	return activeJobs
}

func (node *RequesterNode) triggerStateTransition(ctx context.Context, event model.JobEvent, shard model.JobShard) error {
	ctx, span := node.newSpan(ctx, event.EventName.String())
	defer span.End()

	if shardState, ok := node.shardStateManager.GetShardState(shard); ok {
		switch event.EventName {
		case model.JobEventBid:
			shardState.bid(ctx, event.SourceNodeID)
		case model.JobEventResultsProposed:
			shardState.verifyResult(ctx, event.SourceNodeID)
		case model.JobEventResultsPublished:
			shardState.resultsPublished(ctx, event.SourceNodeID)
		case model.JobEventComputeError:
			shardState.computeError(ctx, event.SourceNodeID)
		}
	} else {
		log.Ctx(ctx).Debug().Msgf("Received %s for unknown shard %s", event.EventName, shard)
		if err := node.notifyShardInvalidRequest(ctx, shard, event.SourceNodeID, "shard state not found"); err != nil {
			log.Ctx(ctx).Warn().Msgf(
				"Received %s for unknown shard %s, and failed to notify the source node %s",
				event.EventName, shard, event.SourceNodeID)
		}
	}
	return nil
}

// called for both JobEventShardCompleted and JobEventShardError
// we ask the verifier "IsExecutionComplete" to decide if we can start
// verifying the results - each verifier might have a different
// answer for IsExecutionComplete so we pass off to it to decide
// we mark the job as "verifying" to prevent duplicate verification
func (node *RequesterNode) verifyShard(
	ctx context.Context,
	shard model.JobShard,
) ([]verifier.VerifierResult, error) {
	jobVerifier, err := node.verifiers.GetVerifier(ctx, shard.Job.Spec.Verifier)
	if err != nil {
		return nil, err
	}
	// ask the verifier if we have enough to start the verification yet
	isExecutionComplete, err := jobVerifier.IsExecutionComplete(ctx, shard)
	if err != nil {
		return nil, err
	}
	if !isExecutionComplete {
		return nil, fmt.Errorf("verifying shard %s but execution is not complete", shard)
	}

	verificationResults, err := jobVerifier.VerifyShard(ctx, shard)
	if err != nil {
		return nil, err
	}

	// we don't fail on first error from the bid queue to avoid a poison pill blocking any progress
	var firstError error
	var verifiedResults []verifier.VerifierResult
	// loop over each verification result and publish events
	for _, verificationResult := range verificationResults {
		notifyErr := node.notifyVerificationResult(ctx, verificationResult)
		if notifyErr != nil && firstError == nil {
			firstError = notifyErr
		} else if verificationResult.Verified {
			verifiedResults = append(verifiedResults, verificationResult)
		}
	}
	if firstError != nil {
		return verifiedResults, firstError
	}

	err = node.notifyVerificationComplete(ctx, shard.Job.ID)
	if err != nil {
		return nil, err
	}

	return verifiedResults, nil
}

// send a job event to notify the compute node that the bid has been accepted or rejected
func (node *RequesterNode) notifyBidDecision(ctx context.Context, shard model.JobShard, targetNodeID string, accepted bool) error {
	jobEventName := model.JobEventBidAccepted
	localEventName := model.JobLocalEventBidAccepted
	if !accepted {
		jobEventName = model.JobEventBidRejected
		localEventName = model.JobLocalEventBidRejected
	}
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with %s for bid: %s", node.ID, jobEventName, shard)

	// publish a local event
	localEvent := model.JobLocalEvent{
		EventName:    localEventName,
		JobID:        shard.Job.ID,
		ShardIndex:   shard.Index,
		TargetNodeID: targetNodeID,
	}
	err := node.localEventConsumer.HandleLocalEvent(ctx, localEvent)
	if err != nil {
		return err
	}

	// the target node is the "nodeID" because the requester node calls this
	// function and so knows which node it is accepting/rejecting the bid for
	jobEvent := node.constructShardEvent(shard, jobEventName)
	jobEvent.TargetNodeID = targetNodeID
	return node.jobEventPublisher.HandleJobEvent(ctx, jobEvent)
}

// send a job event to notify the compute node that the verification has been completed
func (node *RequesterNode) notifyVerificationResult(ctx context.Context, result verifier.VerifierResult) error {
	jobEventName := model.JobEventResultsAccepted
	if !result.Verified {
		jobEventName = model.JobEventResultsRejected
	}
	log.Ctx(ctx).Debug().Msgf("Requester node %s responding with %s results: job=%s node=%s shard=%d",
		node.ID, jobEventName, result.JobID, result.NodeID, result.ShardIndex,
	)

	jobEvent := node.constructJobEvent(result.JobID, jobEventName)
	jobEvent.TargetNodeID = result.NodeID
	jobEvent.ShardIndex = result.ShardIndex
	jobEvent.VerificationResult = model.VerificationResult{
		Complete: true,
		Result:   result.Verified,
	}
	return node.jobEventPublisher.HandleJobEvent(ctx, jobEvent)
}

// local event for requester to know it has already verified this job
func (node *RequesterNode) notifyVerificationComplete(ctx context.Context, jobID string) error {
	return node.localEventConsumer.HandleLocalEvent(ctx, model.JobLocalEvent{
		EventName: model.JobLocalEventVerified,
		JobID:     jobID,
	})
}

func (node *RequesterNode) notifyShardError(
	ctx context.Context,
	shard model.JobShard,
	status string,
) error {
	ev := node.constructShardEvent(shard, model.JobEventError)
	ev.Status = status
	return node.jobEventPublisher.HandleJobEvent(ctx, ev)
}

func (node *RequesterNode) notifyShardInvalidRequest(
	ctx context.Context,
	shard model.JobShard,
	targetNodeID string,
	status string,
) error {
	ev := node.constructShardEvent(shard, model.JobEventInvalidRequest)
	ev.Status = status
	ev.TargetNodeID = targetNodeID
	return node.jobEventPublisher.HandleJobEvent(ctx, ev)
}

func (node *RequesterNode) constructJobEvent(jobID string, eventName model.JobEventType) model.JobEvent {
	return model.JobEvent{
		SourceNodeID: node.ID,
		JobID:        jobID,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
}

func (node *RequesterNode) constructShardEvent(shard model.JobShard, eventName model.JobEventType) model.JobEvent {
	event := node.constructJobEvent(shard.Job.ID, eventName)
	event.ShardIndex = shard.Index
	return event
}

func (node *RequesterNode) newSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	// no need to set nodeID and jobID attributes, as they should already be set by the
	// chained event handler context provider
	return system.Span(ctx, "requester_node/requester_node", name,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

func (node *RequesterNode) newRootSpanForJob(ctx context.Context, jobID string) (context.Context, trace.Span) {
	return system.Span(ctx, "requester_node/requester_node", "JobLifecycle",
		// job lifecycle spans go in their own, dedicated trace
		trace.WithNewRoot(),

		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(model.TracerAttributeNameNodeID, node.ID),
			attribute.String(model.TracerAttributeNameJobID, jobID),
		),
	)
}
