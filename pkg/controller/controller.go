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
	"gotest.tools/gotestsum/log"
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

	// this is the transport subscription
	// we will hear events emitted from other nodes on the network
	transport.Subscribe(func(ctx context.Context, ev executor.JobEvent) { // ignore events that we broadcast because we have already handled the event
		// in the controller method
		if ev.NodeID == ctrl.nodeID {
			return
		}
		err := ctrl.processEvent(ctx, ev)

		if err != nil {
			log.Error().Msgf("%s", err)
		}
	})

	return ctrl, nil
}

/*

  internal event handling

*/

// check the state transition of the job is allowed based on the current local state
func (ctrl *Controller) validateStateTransition(ctx context.Context, ev executor.JobEvent) error {
	return nil
}

// mutate the datastore with the given event
func (ctrl *Controller) mutateDatastore(ctx context.Context, ev executor.JobEvent) error {
	// work out which internal handler function based on the event type
	switch ev.EventName {
	case executor.JobEventCreated:
		return ctrl.handleJobCreated(ctx, ev)
	default:
		return fmt.Errorf("unhandled event type: %s", ev.EventName)
	}
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
//   * mutate the local data store
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

// trigger the local subscriptions of the compute and requestor nodes
// and send the event out to the transport so other nodes hear about it
func (ctrl *Controller) writeEvent(ctx context.Context, ev executor.JobEvent) error {

	ctrl.emitEvent(ctx, ev)

	// tell the rest of the network about the event via the transport
	err := ctrl.transport.Publish(ctx, ev)
	if err != nil {
		return err
	}

	return nil
}

func (ctrl *Controller) constructEvent(jobID string, eventName executor.JobEventType) executor.JobEvent {
	return executor.JobEvent{
		NodeID:    ctrl.nodeID,
		JobID:     jobID,
		EventName: eventName,
		EventTime: time.Now(),
	}
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

	jobCreatedEvent := ctrl.constructEvent(jobID, executor.JobEventCreated)
	jobCreatedEvent.JobSpec = spec
	jobCreatedEvent.JobDeal = deal

	// Get the local datastore updated with this event
	job, ev, err := ctrl.handleJobCreated(jobCtx, jobID, spec, deal)
	if err != nil {
		return job, err
	}

	// tell our local subscribers out the create event
	ctrl.broadcastEvent(jobCtx, ev)

	return job, nil
}

func (ctrl *Controller) UpdateDeal(ctx context.Context, jobID string, deal executor.JobDeal) error {
	ctx = ctrl.getJobNodeContext(ctx, jobID)
	ctrl.addJobLifecycleEvent(ctx, jobID, "write_UpdateDeal")

	ev, err := ctrl.handleDealUpdated(ctx, jobID, deal)
	if err != nil {
		return err
	}

	ev := executor.JobEvent{
		JobID:     jobID,
		EventName: executor.JobEventDealUpdated,
		JobDeal:   deal,
		EventTime: time.Now(),
	}
}
