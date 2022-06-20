package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/google/uuid"
)

type WriteEventHandlerFn func(ctx context.Context, event *executor.JobEvent) error

// a useful generic scheduler that given a function to write a job event
// will look after a lot of the boilerplate on behalf on a scheduler implementation
type GenericTransport struct {
	NodeId string
	Jobs   map[string]*executor.Job
	Mutex  sync.Mutex
	// the list of functions to call when we get an update about a job
	SubscribeFuncs    []func(jobEvent *executor.JobEvent, job *executor.Job)
	WriteEventHandler WriteEventHandlerFn
}

func NewGenericTransport(
	nodeID string,
	writeEventHandler WriteEventHandlerFn,
) *GenericTransport {
	return &GenericTransport{
		NodeId:            nodeID,
		Jobs:              make(map[string]*executor.Job),
		SubscribeFuncs:    []func(jobEvent *executor.JobEvent, job *executor.Job){},
		WriteEventHandler: writeEventHandler,
	}
}

func (transport *GenericTransport) writeEvent(ctx context.Context,
	event *executor.JobEvent) error {

	if event.NodeId == "" {
		event.NodeId = transport.NodeId
	}

	return transport.WriteEventHandler(ctx, event)
}

func (transport *GenericTransport) BroadcastEvent(event *executor.JobEvent) {
	transport.Mutex.Lock()
	defer transport.Mutex.Unlock()

	// let's initialise the state for this job because it was just created

	if _, ok := transport.Jobs[event.JobId]; !ok {
		transport.Jobs[event.JobId] = &executor.Job{
			Id:        event.JobId,
			Owner:     event.NodeId,
			Spec:      nil,
			Deal:      nil,
			State:     make(map[string]*executor.JobState),
			CreatedAt: time.Now(),
		}
	}

	// for "create" and "update" events - this will be filled in
	if event.JobSpec != nil {
		transport.Jobs[event.JobId].Spec = event.JobSpec
	}

	// only the owner of the job can update
	if event.JobDeal != nil {
		transport.Jobs[event.JobId].Deal = event.JobDeal
	}

	// both the jobState struct and the NodeId are required
	// because the job state is "against" the node
	if event.JobState != nil && event.NodeId != "" {
		transport.Jobs[event.JobId].State[event.NodeId] = event.JobState
	}

	for _, subscribeFunc := range transport.SubscribeFuncs {
		go subscribeFunc(event, transport.Jobs[event.JobId])
	}

}

/////////////////////////////////////////////////////////////
/// LIFECYCLE
/////////////////////////////////////////////////////////////

// Start the job scheduler. Not that this is blocking and can be managed
// via the context parameter. You must call Subscribe _before_ starting.
func (transport *GenericTransport) Start(ctx context.Context) error {
	panic("should be implemented by parent transport")
}

// HostID returns a unique string per host in whatever network the
// scheduler is connecting to. Must be unique per instance.
func (transport *GenericTransport) HostID(ctx context.Context) (
	string, error) {

	panic("should be implemented by parent transport")
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (transport *GenericTransport) List(ctx context.Context) (
	ListResponse, error) {

	return ListResponse{
		Jobs: transport.Jobs,
	}, nil
}

func (transport *GenericTransport) Get(ctx context.Context, id string) (
	*executor.Job, error) {

	job, ok := transport.Jobs[id]
	if !ok {
		return nil, fmt.Errorf("Job %s not found", id)
	} else {
		return job, nil
	}
}

func (transport *GenericTransport) Subscribe(ctx context.Context,
	subscribeFunc func(jobEvent *executor.JobEvent, job *executor.Job)) {

	transport.SubscribeFuncs = append(transport.SubscribeFuncs, subscribeFunc)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER NODE
/////////////////////////////////////////////////////////////

func (transport *GenericTransport) SubmitJob(ctx context.Context,
	spec *executor.JobSpec, deal *executor.JobDeal) (*executor.Job, error) {

	jobUuid, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("error creating job id: %w", err)
	}
	jobID := jobUuid.String()

	err = transport.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventCreated,
		JobSpec:   spec,
		JobDeal:   deal,
		EventTime: time.Now(),
	})
	if err != nil {
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

func (transport *GenericTransport) UpdateDeal(ctx context.Context,
	jobID string, deal *executor.JobDeal) error {

	return transport.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventDealUpdated,
		JobDeal:   deal,
		EventTime: time.Now(),
	})
}

func (transport *GenericTransport) CancelJob(ctx context.Context,
	jobID string) error {

	panic("should be implemented by parent transport")
}

func (transport *GenericTransport) AcceptJobBid(ctx context.Context,
	jobID, nodeID string) error {

	job, err := transport.Get(ctx, jobID)
	if err != nil {
		return err
	}
	job.Deal.AssignedNodes = append(job.Deal.AssignedNodes, nodeID)
	return transport.writeEvent(ctx, &executor.JobEvent{
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

func (transport *GenericTransport) RejectJobBid(ctx context.Context,
	jobID, nodeID, message string) error {

	if message == "" {
		message = "Job bid rejected by client."
	}

	return transport.writeEvent(ctx, &executor.JobEvent{
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

func (transport *GenericTransport) BidJob(ctx context.Context,
	jobID string) error {

	return transport.writeEvent(ctx, &executor.JobEvent{
		JobId:     jobID,
		EventName: executor.JobEventBid,
		JobState: &executor.JobState{
			State: executor.JobStateBidding,
		},
		EventTime: time.Now(),
	})
}

func (transport *GenericTransport) SubmitResult(ctx context.Context,
	jobID, status, resultsID string) error {

	return transport.writeEvent(ctx, &executor.JobEvent{
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

func (transport *GenericTransport) ErrorJob(ctx context.Context,
	jobID, status string) error {

	return transport.writeEvent(ctx, &executor.JobEvent{
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
func (transport *GenericTransport) ErrorJobForNode(ctx context.Context,
	jobID, nodeID, status string) error {

	return transport.writeEvent(ctx, &executor.JobEvent{
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

// Compile-time interface check:
var _ Transport = (*GenericTransport)(nil)
