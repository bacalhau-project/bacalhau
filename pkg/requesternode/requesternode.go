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

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RequesterNodeConfig struct{}

type RequesterNode struct {
<<<<<<< HEAD
	id             string
	config         RequesterNodeConfig //nolint:gocritic
	controller     *controller.Controller
	verifiers      map[model.Verifier]verifier.Verifier
	componentMutex sync.Mutex
	bidMutex       sync.Mutex
	verifyMutex    sync.Mutex
||||||| 5d1cca3e
	id             string
	config         RequesterNodeConfig //nolint:gocritic
	controller     *controller.Controller
	verifiers      map[model.VerifierType]verifier.Verifier
	componentMutex sync.Mutex
	bidMutex       sync.Mutex
	verifyMutex    sync.Mutex
=======
	ID                 string
	localDB            localdb.LocalDB
	localEventConsumer eventhandler.LocalEventHandler
	jobEventPublisher  eventhandler.JobEventHandler
	verifiers          map[model.VerifierType]verifier.Verifier
	storageProviders   map[model.StorageSourceType]storage.StorageProvider
	config             RequesterNodeConfig //nolint:gocritic
	componentMutex     sync.Mutex
	bidMutex           sync.Mutex
	verifyMutex        sync.Mutex
>>>>>>> main
}

func NewRequesterNode(
	ctx context.Context,
<<<<<<< HEAD
	cm *system.CleanupManager,
	c *controller.Controller,
	verifiers map[model.Verifier]verifier.Verifier,
||||||| 5d1cca3e
	cm *system.CleanupManager,
	c *controller.Controller,
	verifiers map[model.VerifierType]verifier.Verifier,
=======
	nodeID string,
	localDB localdb.LocalDB,
	localEventConsumer eventhandler.LocalEventHandler,
	jobEventPublisher eventhandler.JobEventHandler,
	verifiers map[model.VerifierType]verifier.Verifier,
	storageProviders map[model.StorageSourceType]storage.StorageProvider,
>>>>>>> main
	config RequesterNodeConfig, //nolint:gocritic
) (*RequesterNode, error) {
	// TODO: instrument with trace
	requesterNode := &RequesterNode{
		ID:                 nodeID,
		localDB:            localDB,
		localEventConsumer: localEventConsumer,
		jobEventPublisher:  jobEventPublisher,
		verifiers:          verifiers,
		storageProviders:   storageProviders,
		config:             config,
	}
	requesterNode.bidMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "RequesterNode.bidMutex",
	})
	requesterNode.verifyMutex.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "RequesterNode.verifyMutex",
	})

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
	case model.JobEventBid:
		return node.handleEventBid(ctx, j, event)
	case model.JobEventResultsProposed:
		return node.handleEventShardExecutionComplete(ctx, j, event)
	case model.JobEventError:
		return node.handleEventShardExecutionComplete(ctx, j, event)
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
	jobCtx, _ := node.newRootSpanForJob(ctx, jobID)

	// TODO: Should replace the span above, with the below, but I don't understand how/why we're tracing contexts in a variable.
	// Specifically tracking them all in ctrl.jobContexts
	// ctx, span := system.NewRootSpan(ctx, system.GetTracer(), "pkg/controller.SubmitJob")
	// defer span.End()

	ev := node.constructJobEvent(jobID, model.JobEventCreated)

	executionPlan, err := jobutils.GenerateExecutionPlan(ctx, data.Job.Spec, node.storageProviders)
	if err != nil {
		return &model.Job{}, fmt.Errorf("error generating execution plan: %s", err)
	}

	ev.ClientID = data.ClientID
	ev.JobSpec = data.Job.Spec
	ev.JobDeal = data.Job.Deal
	ev.JobExecutionPlan = executionPlan

	job := jobutils.ConstructJobFromEvent(ev)
	err = node.localDB.AddJob(ctx, job)
	if err != nil {
		return &model.Job{}, fmt.Errorf("error saving job id: %w", err)
	}

	err = node.jobEventPublisher.HandleJobEvent(jobCtx, ev)
	if err != nil {
		return &model.Job{}, fmt.Errorf("error handling new job event: %s", err)
	}

	return job, nil
}

func (node *RequesterNode) UpdateDeal(ctx context.Context, jobID string, deal model.JobDeal) error {
	ev := node.constructJobEvent(jobID, model.JobEventDealUpdated)
	ev.JobDeal = deal
	return node.jobEventPublisher.HandleJobEvent(ctx, ev)
}

func (node *RequesterNode) handleEventBid(ctx context.Context, j *model.Job, event model.JobEvent) error {
	node.bidMutex.Lock()
	defer node.bidMutex.Unlock()

	// Need to declare span separately to prevent shadowing
	var span trace.Span
	ctx, span = node.newSpan(ctx, "JobEventBid")
	defer span.End()

	bidQueueResults, err := processIncomingBid(ctx, node.localDB, j, event)

	if err != nil {
		return err
	}

	// we don't fail on first error from the bid queue to avoid a poison pill blocking any progress
	var firstError error
	for _, bidQueueResult := range bidQueueResults {
		err := node.notifyBidDecision(ctx, j.ID, event.ShardIndex, bidQueueResult)
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	return firstError
}

// called for both JobEventShardCompleted and JobEventShardError
// we ask the verifier "IsExecutionComplete" to decide if we can start
// verifying the results - each verifier might have a different
// answer for IsExecutionComplete so we pass off to it to decide
// we mark the job as "verifying" to prevent duplicate verification
func (node *RequesterNode) handleEventShardExecutionComplete(
	ctx context.Context,
	j *model.Job,
	jobEvent model.JobEvent,
) error {
	node.verifyMutex.Lock()
	defer node.verifyMutex.Unlock()
	err := node.attemptVerification(ctx, j)
	if err != nil {
		ev := node.constructJobEvent(j.ID, model.JobEventError)
		ev.Status = err.Error()
		ev.ShardIndex = jobEvent.ShardIndex

		errShard := node.jobEventPublisher.HandleJobEvent(ctx, jobEvent)
		if errShard != nil {
			log.Warn().Msgf("ErrorShard failed: %s", errShard.Error())
		}
	}
	return err
}

func (node *RequesterNode) attemptVerification(
	ctx context.Context,
	j *model.Job,
) error {
	jobVerifier, err := node.getVerifier(ctx, j.Spec.Verifier)
	if err != nil {
		return err
	}
	// ask the verifier if we have enough to start the verification yet
	isExecutionComplete, err := jobVerifier.IsExecutionComplete(ctx, j.ID)
	if err != nil {
		return err
	}
	if !isExecutionComplete {
		return nil
	}
	// check that we have not already verified this job
	hasVerified, err := node.localDB.HasLocalEvent(ctx, j.ID, localdb.EventFilterByType(model.JobLocalEventVerified))
	if err != nil {
		return err
	}
	if hasVerified {
		return nil
	}
	verificationResults, err := jobVerifier.VerifyJob(ctx, j.ID)
	if err != nil {
		return err
	}

	// we don't fail on first error from the bid queue to avoid a poison pill blocking any progress
	var firstError error
	// loop over each verification result and publish events
	for _, verificationResult := range verificationResults {
		err := node.notifyVerificationResult(ctx, verificationResult)
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	if firstError != nil {
		return firstError
	}
	return node.notifyVerificationComplete(ctx, j.ID)
}

//nolint:dupl // methods are not duplicates
func (node *RequesterNode) getVerifier(ctx context.Context, typ model.Verifier) (verifier.Verifier, error) {
	node.componentMutex.Lock()
	defer node.componentMutex.Unlock()

	if _, ok := node.verifiers[typ]; !ok {
		return nil, fmt.Errorf(
			"no matching verifier found on this server: %s", typ.String())
	}

	v := node.verifiers[typ]
	installed, err := v.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("verifier is not installed: %s", typ.String())
	}

	return v, nil
}

// send a job event to notify the compute node that the bid has been accepted or rejected
func (node *RequesterNode) notifyBidDecision(ctx context.Context, jobID string, shardIndex int, bidResult bidQueueResult) error {
	jobEventName := model.JobEventBidAccepted
	localEventName := model.JobLocalEventBidAccepted
	if !bidResult.accepted {
		jobEventName = model.JobEventBidRejected
		localEventName = model.JobLocalEventBidRejected
	}
	log.Debug().Msgf("Requester node %s responding with %s for bid: %s %d", node.ID, jobEventName, jobID, shardIndex)

	// publish a local event
	localEvent := model.JobLocalEvent{
		EventName:    localEventName,
		JobID:        jobID,
		TargetNodeID: bidResult.nodeID,
		ShardIndex:   shardIndex,
	}
	err := node.localEventConsumer.HandleLocalEvent(ctx, localEvent)
	if err != nil {
		return err
	}

	// the target node is the "nodeID" because the requester node calls this
	// function and so knows which node it is accepting/rejecting the bid for
	jobEvent := node.constructJobEvent(jobID, jobEventName)
	jobEvent.TargetNodeID = bidResult.nodeID
	jobEvent.ShardIndex = shardIndex
	return node.jobEventPublisher.HandleJobEvent(ctx, jobEvent)
}

// send a job event to notify the compute node that the verification has been completed
func (node *RequesterNode) notifyVerificationResult(ctx context.Context, result verifier.VerifierResult) error {
	jobEventName := model.JobEventResultsAccepted
	if !result.Verified {
		jobEventName = model.JobEventResultsRejected
	}
	log.Debug().Msgf("Requester node %s responding with %s results: job=%s node=%s shard=%d",
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

func (node *RequesterNode) constructJobEvent(jobID string, eventName model.JobEventType) model.JobEvent {
	return model.JobEvent{
		SourceNodeID: node.ID,
		JobID:        jobID,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
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
