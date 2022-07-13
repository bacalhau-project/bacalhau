package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/datastore"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	nodeID          string
	datastore       datastore.DataStore
	transport       transport.Transport2
	jobContexts     map[string]context.Context // total job lifecycle
	jobNodeContexts map[string]context.Context // per-node job lifecycle
	subscribeFuncs  []transport.SubscribeFn2
	contextMutex    sync.RWMutex
	subscribeMutex  sync.Mutex
}

func NewController(
	datastore datastore.DataStore,
	transport transport.Transport2,
) (*Controller, error) {
	nodeID, err := transport.HostID(context.Background())
	if err != nil {
		return nil, err
	}

	ctrl := &Controller{
		nodeID:          nodeID,
		datastore:       datastore,
		transport:       transport,
		jobContexts:     make(map[string]context.Context),
		jobNodeContexts: make(map[string]context.Context),
	}

	// listen for events from other nodes on the network
	transport.Subscribe(func(ctx context.Context, ev executor.JobEvent) { // ignore events that we broadcast because we have already handled the event
		// we have already handled this event locally before braodcasting it
		// so we don't need to process it again
		if ev.SourceNodeID == ctrl.nodeID {
			return
		}

		// process event will:
		//   * validate the state transition
		//   * mutate local state
		//   * call our local subscribers
		err := ctrl.processEvent(ctx, ev)

		if err != nil {
			log.Error().Msgf("%s", err)
		}
	})

	return ctrl, nil
}

/*

  event handling

*/

// check the state transition of the job is allowed based on the current local state
func (ctrl *Controller) validateStateTransition(ctx context.Context, ev executor.JobEvent) error {
	return nil
}

// mutate the datastore with the given event
func (ctrl *Controller) mutateDatastore(ctx context.Context, ev executor.JobEvent) error {
	var err error

	// work out which internal handler function based on the event type
	switch ev.EventName {

	case executor.JobEventCreated:
		err = ctrl.datastore.AddJob(ctx, constructJob(ev))

	case executor.JobEventDealUpdated:
		err = ctrl.datastore.UpdateJobDeal(ctx, ev.JobID, ev.JobDeal)

	default:
		err = fmt.Errorf("unhandled event type: %s", ev.EventName)
	}

	if err != nil {
		return err
	}

	err = ctrl.datastore.AddEvent(ctx, ev.JobID, ev)
	if err != nil {
		return err
	}

	return nil
}

// trigger the local subscriptions of the compute and requestor nodes
func (ctrl *Controller) callLocalSubscribers(ctx context.Context, ev executor.JobEvent) {
	ctrl.subscribeMutex.Lock()
	defer ctrl.subscribeMutex.Unlock()
	for _, fn := range ctrl.subscribeFuncs {
		go fn(ctx, ev)
	}
}

// do these things in this order:
//   * apply the event to the state machine to check validity
//   * mutate the job in the local datastore
//   * add the job event to the local datastore
//   * call our subscribers with the event
func (ctrl *Controller) processEvent(ctx context.Context, ev executor.JobEvent) error {
	err := ctrl.validateStateTransition(ctx, ev)
	if err != nil {
		return fmt.Errorf("error validateStateTransition: %s", err)
	}

	err = ctrl.mutateDatastore(ctx, ev)
	if err != nil {
		return fmt.Errorf("error mutateDatastore: %s", err)
	}

	// now trigger our local subscribers with this event
	ctrl.callLocalSubscribers(ctx, ev)

	return nil
}

// first process the event locally and then broadcast it to the network
func (ctrl *Controller) writeEvent(ctx context.Context, ev executor.JobEvent) error {

	// process the event locally
	err := ctrl.processEvent(ctx, ev)
	if err != nil {
		return err
	}

	// tell the rest of the network about the event via the transport
	err = ctrl.transport.Publish(ctx, ev)
	if err != nil {
		return err
	}

	return nil
}

/*

  public API

*/

// called by compute nodes and requestor nodes
// they will hear about job events once the datastore has been updated
func (ctrl *Controller) Subscribe(fn transport.SubscribeFn2) {
	ctrl.subscribeMutex.Lock()
	defer ctrl.subscribeMutex.Unlock()
	ctrl.subscribeFuncs = append(ctrl.subscribeFuncs, fn)
}

func (ctrl *Controller) SubmitJob(ctx context.Context, spec executor.JobSpec, deal executor.JobDeal) (executor.Job, error) {
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
	ev.JobSpec = spec
	ev.JobDeal = deal

	err = ctrl.writeEvent(jobCtx, ev)

	return constructJob(ev), err
}

// can only be done by the requestor node that is responsible for the job
func (ctrl *Controller) UpdateDeal(ctx context.Context, jobID string, deal executor.JobDeal) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_UpdateDeal")
	ev := ctrl.constructEvent(jobID, executor.JobEventDealUpdated)
	ev.JobDeal = deal
	return ctrl.writeEvent(jobCtx, ev)
}

// done by compute nodes when they hear about the job
func (ctrl *Controller) BidJob(ctx context.Context, jobID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_BidJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventBid)
	return ctrl.writeEvent(jobCtx, ev)
}

// can only be done by the requestor node that is responsible for the job
func (ctrl *Controller) AcceptJobBid(ctx context.Context, jobID, nodeID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_AcceptJobBid")
	ev := ctrl.constructEvent(jobID, executor.JobEventBidAccepted)
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

// can only be done by the requestor node that is responsible for the job
func (ctrl *Controller) RejectJobBid(ctx context.Context, jobID, nodeID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_RejectJobBid")
	ev := ctrl.constructEvent(jobID, executor.JobEventBidRejected)
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

// called by a compute node who has already bid
func (ctrl *Controller) CancelJobBid(ctx context.Context, jobID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_CancelJobBid")
	ev := ctrl.constructEvent(jobID, executor.JobEventBidCancelled)
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) PrepareJob(ctx context.Context, jobID, status string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_PrepareJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventPreparing)
	ev.Status = status
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) RunJob(ctx context.Context, jobID, status string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_RunJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventRunning)
	ev.Status = status
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) CompleteJob(ctx context.Context, jobID, status, resultsID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_CompleteJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventCompleted)
	ev.Status = status
	ev.ResultsID = resultsID
	return ctrl.writeEvent(jobCtx, ev)
}

// can only be called by a compute node who is current assigned to the job
func (ctrl *Controller) ErrorJob(ctx context.Context, jobID, status, resultsID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_ErrorJob")
	ev := ctrl.constructEvent(jobID, executor.JobEventError)
	ev.Status = status
	ev.ResultsID = resultsID
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) AcceptResults(ctx context.Context, jobID, nodeID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_AcceptResults")
	ev := ctrl.constructEvent(jobID, executor.JobEventResultsAccepted)
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

func (ctrl *Controller) RejectResults(ctx context.Context, jobID, nodeID string) error {
	jobCtx := ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(jobCtx, jobID, "write_RejectResults")
	ev := ctrl.constructEvent(jobID, executor.JobEventResultsRejected)
	ev.TargetNodeID = nodeID
	return ctrl.writeEvent(jobCtx, ev)
}

/*

  internal helpers

*/

func (ctrl *Controller) constructEvent(jobID string, eventName executor.JobEventType) executor.JobEvent {
	return executor.JobEvent{
		SourceNodeID: ctrl.nodeID,
		JobID:        jobID,
		EventName:    eventName,
		EventTime:    time.Now(),
	}
}

func constructJob(ev executor.JobEvent) executor.Job {
	return executor.Job{
		ID:        ev.JobID,
		Spec:      ev.JobSpec,
		Deal:      ev.JobDeal,
		State:     map[string]executor.JobState{},
		CreatedAt: time.Now(),
	}
}
