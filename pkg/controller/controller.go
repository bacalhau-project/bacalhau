package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Controller struct {
	cleanupManager   *system.CleanupManager
	id               string
	localdb          localdb.LocalDB
	transport        transport.Transport
	storageProviders map[storage.StorageSourceType]storage.StorageProvider
	jobContexts      map[string]context.Context // total job lifecycle
	jobNodeContexts  map[string]context.Context // per-node job lifecycle
	subscribeFuncs   []transport.SubscribeFn
	contextMutex     sync.RWMutex
	subscribeMutex   sync.RWMutex
}

/*

  lifecycle

*/

func NewController(
	cm *system.CleanupManager,
	db localdb.LocalDB,
	tx transport.Transport,
	storageProviders map[storage.StorageSourceType]storage.StorageProvider,
) (*Controller, error) {
	nodeID, err := tx.HostID(context.Background())
	if err != nil {
		return nil, err
	}
	ctrl := &Controller{
		cleanupManager:   cm,
		id:               nodeID,
		localdb:          db,
		transport:        tx,
		storageProviders: storageProviders,
		jobContexts:      make(map[string]context.Context),
		jobNodeContexts:  make(map[string]context.Context),
	}

	return ctrl, nil
}

func (ctrl *Controller) GetTransport() transport.Transport {
	return ctrl.transport
}

func (ctrl *Controller) GetLocalDB() localdb.LocalDB {
	return ctrl.localdb
}

func (ctrl *Controller) Start(ctx context.Context) error {
	ctrl.transport.Subscribe(func(ctx context.Context, ev executor.JobEvent) {
		err := ctrl.handleEvent(ctx, ev)
		if err != nil {
			log.Error().Msgf("error in handle event: %s\n%+v", err, ev)
		}
	})

	ctrl.cleanupManager.RegisterCallback(func() error {
		return ctrl.Shutdown(ctx)
	})

	return ctrl.transport.Start(ctx)
}

func (ctrl *Controller) Shutdown(ctx context.Context) error {
	return ctrl.cleanJobContexts(ctx)
}

func (ctrl *Controller) HostID(ctx context.Context) (string, error) {
	return ctrl.id, nil
}

// called by compute nodes and requestor nodes
// they will hear about job events once the datastore has been updated
func (ctrl *Controller) Subscribe(fn transport.SubscribeFn) {
	ctrl.subscribeMutex.Lock()
	defer ctrl.subscribeMutex.Unlock()
	ctrl.subscribeFuncs = append(ctrl.subscribeFuncs, fn)
}

/*

  READ API

*/
func (ctrl *Controller) GetJob(ctx context.Context, id string) (executor.Job, error) {
	return ctrl.localdb.GetJob(ctx, id)
}

func (ctrl *Controller) GetJobEvents(ctx context.Context, id string) ([]executor.JobEvent, error) {
	return ctrl.localdb.GetJobEvents(ctx, id)
}

func (ctrl *Controller) GetJobLocalEvents(ctx context.Context, id string) ([]executor.JobLocalEvent, error) {
	return ctrl.localdb.GetJobLocalEvents(ctx, id)
}

// we return an array of job states against each job - one for each shard
func (ctrl *Controller) GetJobState(ctx context.Context, id string) (executor.JobState, error) {
	return ctrl.localdb.GetJobState(ctx, id)
}

func (ctrl *Controller) GetJobs(ctx context.Context, query localdb.JobQuery) ([]executor.Job, error) {
	return ctrl.localdb.GetJobs(ctx, query)
}

/*

  REQUESTER NODE

*/
func (ctrl *Controller) SubmitJob(
	ctx context.Context,
	data executor.JobCreatePayload,
) (executor.Job, error) {
	jobUUID, err := uuid.NewRandom()
	if err != nil {
		return executor.Job{}, fmt.Errorf("error creating job id: %w", err)
	}
	jobID := jobUUID.String()

	// Creates a new root context to track a job's lifecycle for tracing. This
	// should be fine as only one node will call SubmitJob(...) - the other
	// nodes will hear about the job via events on the transport.
	jobCtx, _ := ctrl.newRootSpanForJob(ctx, jobID)

	ev := ctrl.constructEvent(jobID, executor.JobEventCreated)

	ev.ClientID = data.ClientID
	ev.JobSpec = data.Spec
	ev.JobDeal = data.Deal

	job := jobutils.ConstructJobFromEvent(ev)

	job, err = jobutils.ProcessJobSharding(ctx, job, ctrl.storageProviders)
	if err != nil {
		return executor.Job{}, fmt.Errorf("error processing job sharding: %s", err)
	}

	fmt.Printf("job --------------------------------------\n")
	spew.Dump(job)

	// first write the job to our local data store
	// so clients have consistency when they ask for the job by id
	err = ctrl.localdb.AddJob(ctx, job)
	if err != nil {
		return executor.Job{}, fmt.Errorf("error saving job id: %w", err)
	}

	err = ctrl.writeEvent(jobCtx, ev)
	return job, err
}

// can only be done by the requestor node that is responsible for the job
func (ctrl *Controller) UpdateDeal(ctx context.Context, jobID string, deal executor.JobDeal) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_UpdateDeal")
	ev := ctrl.constructEvent(jobID, executor.JobEventDealUpdated)
	ev.JobDeal = deal
	return ctrl.writeEvent(jobCtx, ev)
}

// can only be done by the requestor node that is responsible for the job
func (ctrl *Controller) AcceptJobBid(ctx context.Context, jobID, nodeID string) error {
	if jobID == "" {
		return fmt.Errorf("AcceptJobBid: jobID cannot be empty")
	}
	if nodeID == "" {
		return fmt.Errorf("AcceptJobBid: nodeID cannot be empty")
	}
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	err := ctrl.localdb.AddLocalEvent(jobCtx, jobID, executor.JobLocalEvent{
		EventName:    executor.JobLocalEventBidAccepted,
		JobID:        jobID,
		TargetNodeID: nodeID,
	})
	if err != nil {
		return err
	}
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_AcceptJobBid")
	ev := ctrl.constructEvent(jobID, executor.JobEventBidAccepted)
	// the target node is the "nodeID" because the requester node calls this
	// function and so knows which node it is accepting the bid for
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

// can only be done by the requestor node that is responsible for the job
func (ctrl *Controller) RejectJobBid(ctx context.Context, jobID, nodeID string) error {
	if jobID == "" {
		return fmt.Errorf("RejectJobBid: jobID cannot be empty")
	}
	if nodeID == "" {
		return fmt.Errorf("RejectJobBid: nodeID cannot be empty")
	}
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_RejectJobBid")
	ev := ctrl.constructEvent(jobID, executor.JobEventBidRejected)
	// the target node is the "nodeID" because the requester node calls this
	// function and so knows which node it is rejecting the bid for
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) AcceptResults(ctx context.Context, jobID, nodeID string) error {
	if jobID == "" {
		return fmt.Errorf("AcceptResults: jobID cannot be empty")
	}
	if nodeID == "" {
		return fmt.Errorf("AcceptResults: nodeID cannot be empty")
	}
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_AcceptResults")
	ev := ctrl.constructEvent(jobID, executor.JobEventResultsAccepted)
	// the target node is the "nodeID" because the requester node calls this
	// function and so knows which node it is rejecting the bid for
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) RejectResults(ctx context.Context, jobID, nodeID string) error {
	if jobID == "" {
		return fmt.Errorf("RejectResults: jobID cannot be empty")
	}
	if nodeID == "" {
		return fmt.Errorf("RejectResults: nodeID cannot be empty")
	}
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_RejectResults")
	ev := ctrl.constructEvent(jobID, executor.JobEventResultsRejected)
	// the target node is the "nodeID" because the requester node calls this
	// function and so knows which node it is rejecting the bid for
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

/*

  COMPUTE NODE

*/

// done by compute nodes when they hear about the job
func (ctrl *Controller) BidJob(ctx context.Context, jobID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	err := ctrl.localdb.AddLocalEvent(jobCtx, jobID, executor.JobLocalEvent{
		EventName: executor.JobLocalEventBid,
		JobID:     jobID,
	})
	if err != nil {
		return err
	}
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_BidJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventBid)
	// the target node is "us" because it is "us" who is bidding
	// and so the job state should be updated against our node id
	ev.TargetNodeID = ctrl.id
	return ctrl.writeEvent(jobCtx, ev)
}

// called by a compute node who has already bid
func (ctrl *Controller) CancelJobBid(ctx context.Context, jobID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_CancelJobBid")
	ev := ctrl.constructEvent(jobID, executor.JobEventBidCancelled)
	// the target node is "us" because it is "us" who is canceling our bid
	// and so the job state should be updated against our node id
	ev.TargetNodeID = ctrl.id
	return ctrl.writeEvent(jobCtx, ev)
}

// this can be used both to indicate the job has started to run
// and also to update the status half way through running it
func (ctrl *Controller) RunJob(ctx context.Context, jobID, status string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_RunJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventRunning)
	ev.Status = status
	// the target node is "us" because it is "us" who is running the job
	// and so the job state should be updated against our node id
	ev.TargetNodeID = ctrl.id
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) CompleteJob(ctx context.Context, jobID, status, resultsID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_CompleteJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventCompleted)
	ev.Status = status
	ev.ResultsID = resultsID
	// the target node is "us" because it is "us" who has completed the job
	// and so the job state should be updated against our node id
	ev.TargetNodeID = ctrl.id
	return ctrl.writeEvent(jobCtx, ev)
}

// can only be called by a compute node who is current assigned to the job
func (ctrl *Controller) ErrorJob(ctx context.Context, jobID, status, resultsID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_ErrorJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventError)
	ev.Status = status
	ev.ResultsID = resultsID
	// the target node is "us" because it is "us" who has errored the job
	// and so the job state should be updated against our node id
	ev.TargetNodeID = ctrl.id
	return ctrl.writeEvent(jobCtx, ev)
}

/*

  MISC FUNCTIONS

*/

// write the "context" for a job to storage
// this is used to upload code files
// we presently just fix on ipfs to do this
func (ctrl *Controller) PinContext(ctx context.Context, buildContext string) (string, error) {
	ipfsStorage := ctrl.storageProviders[storage.StorageSourceIPFS]
	result, err := ipfsStorage.Upload(ctx, buildContext)
	if err != nil {
		return "", err
	}
	return result.Cid, nil
}

/*

  event handlers

*/

// tell the rest of the network about the event via the transport
func (ctrl *Controller) writeEvent(ctx context.Context, ev executor.JobEvent) error {
	jobCtx := ctrl.getJobNodeContext(ctx, ev.JobID)
	return ctrl.transport.Publish(jobCtx, ev)
}

func (ctrl *Controller) handleEvent(ctx context.Context, ev executor.JobEvent) error {
	jobCtx := ctrl.getEventJobContext(ctx, ev)

	err := ctrl.mutateDatastore(jobCtx, ev)
	if err != nil {
		return fmt.Errorf("error mutateDatastore: %s", err)
	}

	// now trigger our local subscribers with this event
	ctrl.callLocalSubscribers(jobCtx, ev)

	log.Trace().Msgf("handleEvent: %+v", ev)

	return nil
}

/*

  process event helpers

*/

// mutate the datastore with the given event
func (ctrl *Controller) mutateDatastore(ctx context.Context, ev executor.JobEvent) error {
	var err error

	// work out which internal handler function based on the event type
	switch ev.EventName {
	case executor.JobEventCreated:
		err = ctrl.localdb.AddJob(ctx, jobutils.ConstructJobFromEvent(ev))

	case executor.JobEventDealUpdated:
		err = ctrl.localdb.UpdateJobDeal(ctx, ev.JobID, ev.JobDeal)
	}

	if err != nil {
		return err
	}

	err = ctrl.localdb.AddEvent(ctx, ev.JobID, ev)
	if err != nil {
		return err
	}

	executionState := executor.GetStateFromEvent(ev.EventName)
	if ev.TargetNodeID != "" && executor.IsValidJobState(executionState) {
		// update the state for this job shard
		err = ctrl.localdb.UpdateShardState(
			ctx,
			ev.JobID,
			ev.TargetNodeID,
			ev.ShardIndex,
			executor.JobShardState{
				NodeID:     ev.TargetNodeID,
				ShardIndex: ev.ShardIndex,
				State:      executionState,
				Status:     ev.Status,
				ResultsID:  ev.ResultsID,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// trigger the local subscriptions of the compute and requestor nodes
// we run them in parallel but block on them all finishing
// otherwise the context would be canceled
func (ctrl *Controller) callLocalSubscribers(ctx context.Context, ev executor.JobEvent) {
	ctrl.subscribeMutex.RLock()
	defer ctrl.subscribeMutex.RUnlock()

	// run all local subscribers in parallel
	var wg sync.WaitGroup
	for _, fn := range ctrl.subscribeFuncs {
		wg.Add(1)
		go func(f transport.SubscribeFn) {
			defer wg.Done()
			f(ctx, ev)
		}(fn)
	}
	wg.Wait()
}

/*

  utils

*/

func (ctrl *Controller) constructEvent(jobID string, eventName executor.JobEventType) executor.JobEvent {
	return executor.JobEvent{
		SourceNodeID: ctrl.id,
		JobID:        jobID,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
}

/*

  otel

*/

func (ctrl *Controller) getEventJobContext(ctx context.Context, ev executor.JobEvent) context.Context {
	jobCtx := ctrl.getJobNodeContext(ctx, ev.JobID)

	ctrl.addJobLifecycleEvent(jobCtx, ev.JobID, fmt.Sprintf("read_%s", ev.EventName))

	// If the event is known to be ignorable, end the local lifecycle context:
	if ev.EventName.IsIgnorable() {
		ctrl.endJobNodeContext(ev.JobID)
	}

	// If the event is known to be terminal, end the global lifecycle context:
	if ev.EventName.IsTerminal() {
		ctrl.endJobContext(ev.JobID)
	}

	return jobCtx
}

func (ctrl *Controller) cleanJobContexts(ctx context.Context) error {
	ctrl.contextMutex.RLock()
	defer ctrl.contextMutex.RUnlock()
	// End all job lifecycle spans so we don't lose any tracing data:
	for _, ctx := range ctrl.jobContexts {
		trace.SpanFromContext(ctx).End()
	}
	for _, ctx := range ctrl.jobNodeContexts {
		trace.SpanFromContext(ctx).End()
	}

	return nil
}

// endJobContext ends the global lifecycle context for a job.
func (ctrl *Controller) endJobContext(jobID string) {
	ctx := ctrl.getJobContext(jobID)
	trace.SpanFromContext(ctx).End()
	ctrl.contextMutex.Lock()
	defer ctrl.contextMutex.Unlock()
	delete(ctrl.jobContexts, jobID)
}

// endJobNodeContext ends the local lifecycle context for a job.
func (ctrl *Controller) endJobNodeContext(jobID string) {
	ctx := ctrl.getJobNodeContext(context.Background(), jobID)
	trace.SpanFromContext(ctx).End()
	ctrl.contextMutex.Lock()
	defer ctrl.contextMutex.Unlock()
	delete(ctrl.jobNodeContexts, jobID)
}

// getJobContext returns a context that tracks the global lifecycle of a job
// as it is processed by this and other nodes in the bacalhau network.
func (ctrl *Controller) getJobContext(jobID string) context.Context {
	ctrl.contextMutex.RLock()
	defer ctrl.contextMutex.RUnlock()
	jobCtx, ok := ctrl.jobContexts[jobID]
	if !ok {
		return context.Background() // no lifecycle context yet
	}
	return jobCtx
}

// getJobNodeContext returns a context that tracks the local lifecycle of a
// job as it has been processed by this node.
func (ctrl *Controller) getJobNodeContext(ctx context.Context, jobID string) context.Context {
	ctrl.contextMutex.Lock()
	defer ctrl.contextMutex.Unlock()

	jobCtx, _ := system.Span(ctx, "controller",
		"JobLifecycle-"+ctrl.id[:8],
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("jobID", jobID),
			attribute.String("nodeID", ctrl.id),
		),
	)

	ctrl.jobNodeContexts[jobID] = jobCtx
	return jobCtx
}

func (ctrl *Controller) addJobLifecycleEvent(ctx context.Context, jobID, eventName string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(eventName,
		trace.WithAttributes(
			append(attrs,
				attribute.String("jobID", jobID),
				attribute.String("nodeID", ctrl.id),
			)...,
		),
	)
}

func (ctrl *Controller) newRootSpanForJob(ctx context.Context, jobID string) (context.Context, trace.Span) {
	jobCtx, span := system.Span(ctx, "controller", "JobLifecycle",
		// job lifecycle spans go in their own, dedicated trace
		trace.WithNewRoot(),

		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("jobID", jobID),
			attribute.String("nodeID", ctrl.id),
		),
	)

	ctrl.contextMutex.Lock()
	defer ctrl.contextMutex.Unlock()
	ctrl.jobContexts[jobID] = jobCtx

	return jobCtx, span
}
