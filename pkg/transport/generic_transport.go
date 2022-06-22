package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// WriteEventHandlerFn is provided by a parent transport layer for broadcasting
// messages to other bacalhau nodes.
type WriteEventHandlerFn func(context.Context, *executor.JobEvent) error

// GenericTransport is a generic base transport layer that handles a bunch of
// boilerplate for a parent transport. The parent transport just has to provide
// a WriteEventHandlerFn for broadcasting messages to other bacalhau nodes, and
// call the appropriate GenericTransport functions when messages are received
// from other bacalhau nodes.
type GenericTransport struct {
	// NodeID is the ID of the compute node running this transport.
	NodeID string

	// SubscribeFuncs contains callbacks for in-process event listeners.
	SubscribeFuncs []SubscribeFn

	// Internal state:
	jobs              map[string]*executor.Job   // list of known jobs
	jobContexts       map[string]context.Context // total job lifecycle
	jobNodeContexts   map[string]context.Context // per-node job lifecycle
	mutex             sync.Mutex                 // thread-safety for maps
	writeEventHandler WriteEventHandlerFn        // parent transport callback
}

func NewGenericTransport(nodeID string,
	writeEventHandler WriteEventHandlerFn) *GenericTransport {

	return &GenericTransport{
		NodeID:         nodeID,
		SubscribeFuncs: []SubscribeFn{},

		jobs:              make(map[string]*executor.Job),
		jobContexts:       make(map[string]context.Context),
		jobNodeContexts:   make(map[string]context.Context),
		writeEventHandler: writeEventHandler,
	}
}

// writeEvent calls the parent transport's WriteEventHandler, which should
// broadcast the event to its distributed network of bacalhau nodes.
func (gt *GenericTransport) writeEvent(ctx context.Context,
	event *executor.JobEvent) error {

	if event.NodeId == "" {
		event.NodeId = gt.NodeID
	}

	return gt.writeEventHandler(ctx, event)
}

// BroadcastEvent notifies every listener in the transport's process of a
// new event. Note that this is purely local, and doesn't broadcast the
// event to the parent transport's network of bacalhau nodes.
func (gt *GenericTransport) BroadcastEvent(ctx context.Context,
	event *executor.JobEvent) {

	gt.mutex.Lock()
	defer gt.mutex.Unlock()

	// Keep track of the state of jobs we hear about:
	if _, ok := gt.jobs[event.JobId]; !ok {
		gt.jobs[event.JobId] = &executor.Job{
			Id:        event.JobId,
			Owner:     event.NodeId,
			Spec:      nil,
			Deal:      nil,
			State:     make(map[string]*executor.JobState),
			CreatedAt: time.Now(),
		}
	}

	// Passed in for create and update events:
	if event.JobSpec != nil {
		gt.jobs[event.JobId].Spec = event.JobSpec
	}

	// Keep track of job owner so we know who can edit a job:
	if event.JobDeal != nil {
		gt.jobs[event.JobId].Deal = event.JobDeal
	}

	// Jobs have different states on different nodes:
	if event.JobState != nil && event.NodeId != "" {
		gt.jobs[event.JobId].State[event.NodeId] = event.JobState
	}

	// Actually notify in-process listeners:
	jobCtx := gt.getJobNodeContext(ctx, event.JobId)
	for _, subscribeFunc := range gt.SubscribeFuncs {
		go subscribeFunc(jobCtx, event, gt.jobs[event.JobId])
	}

}

/////////////////////////////////////////////////////////////
/// LIFECYCLE
/////////////////////////////////////////////////////////////

func (gt *GenericTransport) Start(ctx context.Context) error {
	panic("should be implemented by parent transport")
}

func (gt *GenericTransport) Shutdown(ctx context.Context) error {
	// End all job lifecycle spans so we don't lose any tracing data:
	for _, ctx := range gt.jobContexts {
		trace.SpanFromContext(ctx).End()
	}
	for _, ctx := range gt.jobNodeContexts {
		trace.SpanFromContext(ctx).End()
	}

	return nil
}

func (gt *GenericTransport) HostID(ctx context.Context) (
	string, error) {

	panic("should be implemented by parent transport")
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (gt *GenericTransport) List(ctx context.Context) (
	ListResponse, error) {

	return ListResponse{
		Jobs: gt.jobs,
	}, nil
}

func (gt *GenericTransport) Get(ctx context.Context, id string) (
	*executor.Job, error) {

	job, ok := gt.jobs[id]
	if !ok {
		return nil, fmt.Errorf("job not found in transport: %s", id)
	}

	return job, nil
}

func (gt *GenericTransport) Subscribe(ctx context.Context, fn SubscribeFn) {
	gt.SubscribeFuncs = append(gt.SubscribeFuncs, fn)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER NODE
/////////////////////////////////////////////////////////////

func (gt *GenericTransport) SubmitJob(ctx context.Context,
	spec *executor.JobSpec, deal *executor.JobDeal) (*executor.Job, error) {

	jobUuid, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("error creating job id: %w", err)
	}
	jobID := jobUuid.String()

	// Creates a new root context to track a job's lifecycle for tracing. This
	// should be fine as only one node will call SubmitJob(...) - the other
	// nodes will hear about the job via events on the transport.
	jobCtx, _ := gt.newRootSpanForJob(ctx, jobID)
	gt.jobContexts[jobID] = jobCtx

	if err := gt.writeEvent(jobCtx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventCreated,
		JobSpec:   spec,
		JobDeal:   deal,
		EventTime: time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("error writing job event: %w", err)
	}

	return &executor.Job{
		Id:        jobID,
		Spec:      spec,
		Deal:      deal,
		State:     make(map[string]*executor.JobState),
		CreatedAt: time.Now(),
	}, nil
}

func (gt *GenericTransport) UpdateDeal(ctx context.Context,
	jobID string, deal *executor.JobDeal) error {

	ctx = gt.getJobNodeContext(ctx, jobID)
	gt.addJobLifecycleEvent(ctx, jobID, "UpdateDeal")

	return gt.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventDealUpdated,
		JobDeal:   deal,
		EventTime: time.Now(),
	})
}

func (gt *GenericTransport) CancelJob(ctx context.Context,
	jobID string) error {

	panic("should be implemented by parent transport")
}

func (gt *GenericTransport) AcceptJobBid(ctx context.Context,
	jobID, nodeID string) error {

	ctx = gt.getJobNodeContext(ctx, jobID)
	gt.addJobLifecycleEvent(ctx, jobID, "AcceptJobBid")

	job, err := gt.Get(ctx, jobID)
	if err != nil {
		return err
	}

	job.Deal.AssignedNodes = append(job.Deal.AssignedNodes, nodeID)
	return gt.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		NodeId:    nodeID,
		EventName: executor.JobEventBidAccepted,
		JobDeal:   job.Deal,
		JobState: &executor.JobState{
			State: executor.JobStateRunning,
		},
		EventTime: time.Now(),
	})
}

func (gt *GenericTransport) RejectJobBid(ctx context.Context,
	jobID, nodeID, message string) error {

	ctx = gt.getJobNodeContext(ctx, jobID)
	gt.addJobLifecycleEvent(ctx, jobID, "RejectJobBid") // TODO: add msg

	if message == "" {
		message = "Job bid rejected by client."
	}

	return gt.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		NodeId:    nodeID,
		EventName: executor.JobEventBidRejected,
		JobState: &executor.JobState{
			State:  executor.JobStateBidRejected,
			Status: message,
		},
		EventTime: time.Now(),
	})
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (gt *GenericTransport) BidJob(ctx context.Context,
	jobID string) error {

	ctx = gt.getJobNodeContext(ctx, jobID)
	gt.addJobLifecycleEvent(ctx, jobID, "BidJob")

	return gt.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventBid,
		JobState: &executor.JobState{
			State: executor.JobStateBidding,
		},
		EventTime: time.Now(),
	})
}

func (gt *GenericTransport) SubmitResult(_ context.Context,
	jobID, status, resultsID string) error {

	// TODO: is this really terminal?
	ctx := gt.endJobContext(jobID, "SubmitResult")

	return gt.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventResults,
		JobState: &executor.JobState{
			State:     executor.JobStateComplete,
			Status:    status,
			ResultsId: resultsID,
		},
		EventTime: time.Now(),
	})
}

func (gt *GenericTransport) ErrorJob(_ context.Context,
	jobID, status string) error {

	// TODO: is this really terminal?
	ctx := gt.endJobContext(jobID, "ErrorJob")

	return gt.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventError,
		JobState: &executor.JobState{
			State:  executor.JobStateError,
			Status: status,
		},
		EventTime: time.Now(),
	})
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (gt *GenericTransport) ErrorJobForNode(_ context.Context,
	jobID, nodeID, status string) error {

	// TODO: is this really terminal?
	ctx := gt.endJobContext(jobID, "ErrorJobForNode")

	return gt.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		NodeId:    nodeID,
		EventName: executor.JobEventError,
		JobState: &executor.JobState{
			State:  executor.JobStateError,
			Status: status,
		},
		EventTime: time.Now(),
	})
}

// endJobContext ends the local and global lifecycle contexts for a job.
func (gt *GenericTransport) endJobContext(
	jobID, eventName string) context.Context {

	ctx := gt.getJobNodeContext(context.Background(), jobID)
	gt.addJobLifecycleEvent(ctx, jobID, eventName)
	trace.SpanFromContext(ctx).End() // end the local lifecycle context

	ctx = gt.getJobContext(jobID)
	trace.SpanFromContext(ctx).End() // end the global lifecycle context

	return ctx
}

// getJobContext returns a context that tracks the global lifecycle of a job
// as it is processed by this and other nodes in the bacalhau network.
func (gt *GenericTransport) getJobContext(
	jobID string) context.Context {

	jobCtx, ok := gt.jobContexts[jobID]
	if !ok {
		return context.Background() // no lifecycle context yet
	}
	return jobCtx
}

// getJobNodeContext returns a context that tracks the local lifecycle of a
// job as it has been processed by this node.
func (gt *GenericTransport) getJobNodeContext(ctx context.Context,
	jobID string) context.Context {

	jobCtx, ok := gt.jobNodeContexts[jobID]
	if !ok {
		jobCtx, _ = system.Span(ctx, "transport/generic_transport",
			"JobLifecycle-"+gt.NodeID[:8],
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("job_id", jobID),
				attribute.String("node_id", gt.NodeID),
			),
		)

		gt.jobNodeContexts[jobID] = jobCtx
	}
	return jobCtx
}

func (gt *GenericTransport) addJobLifecycleEvent(
	ctx context.Context, jobID string, eventName string) {

	span := trace.SpanFromContext(ctx)
	span.AddEvent(eventName,
		trace.WithAttributes(
			attribute.String("jobID", jobID),
			attribute.String("nodeID", gt.NodeID),
		),
	)
}

func (gt *GenericTransport) newRootSpanForJob(ctx context.Context,
	jobID string) (context.Context, trace.Span) {

	return system.Span(ctx, "transport/generic_transport", "JobLifecycle",
		// job lifecycle spans go in their own, dedicated trace
		trace.WithNewRoot(),

		trace.WithLinks(trace.LinkFromContext(ctx)), // link to any api traces
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("jobID", jobID),
			attribute.String("nodeID", gt.NodeID),
		),
	)
}

// Compile-time interface check:
var _ Transport = (*GenericTransport)(nil)
