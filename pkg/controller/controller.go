package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/datastore"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	cm              *system.CleanupManager
	id              string
	datastore       datastore.DataStore
	transport       transport.Transport
	jobContexts     map[string]context.Context // total job lifecycle
	jobNodeContexts map[string]context.Context // per-node job lifecycle
	subscribeFuncs  []transport.SubscribeFn
	contextMutex    sync.RWMutex
	subscribeMutex  sync.Mutex
}

/*

  lifecycle

*/

func NewController(
	cm *system.CleanupManager,
	datastore datastore.DataStore,
	transport transport.Transport,
) (*Controller, error) {
	nodeID, err := transport.HostID(context.Background())
	if err != nil {
		return nil, err
	}
	ctrl := &Controller{
		cm:              cm,
		id:              nodeID,
		datastore:       datastore,
		transport:       transport,
		jobContexts:     make(map[string]context.Context),
		jobNodeContexts: make(map[string]context.Context),
	}

	return ctrl, nil
}

func (ctrl *Controller) HostID(ctx context.Context) (string, error) {
	return ctrl.id, nil
}

func (ctrl *Controller) GetJob(ctx context.Context, id string) (datastore.Job, error) {
	return ctrl.datastore.GetJob(ctx, id)
}

func (ctrl *Controller) GetJobs(ctx context.Context, query datastore.JobQuery) ([]datastore.Job, error) {
	return ctrl.datastore.GetJobs(ctx, query)
}

func (ctrl *Controller) Start(ctx context.Context) error {
	// listen for events from other nodes on the network
	ctrl.transport.Subscribe(func(ctx context.Context, ev executor.JobEvent) { // ignore events that we broadcast because we have already handled the event
		// we have already handled this event locally before braodcasting it
		// so we don't need to process it again
		if ev.SourceNodeID == ctrl.id {
			return
		}

		// process event will:
		//   * validate the state transition
		//   * mutate local state
		//   * call our local subscribers
		err := ctrl.handleEvent(ctx, ev)

		if err != nil {
			log.Error().Msgf("%s", err)
		}
	})

	ctrl.cm.RegisterCallback(func() error {
		return ctrl.Shutdown(ctx)
	})

	return ctrl.transport.Start(ctx)
}

func (ctrl *Controller) Shutdown(ctx context.Context) error {
	return ctrl.cleanJobContexts(ctx)
}

/*

  public API

*/

// called by compute nodes and requestor nodes
// they will hear about job events once the datastore has been updated
func (ctrl *Controller) Subscribe(fn transport.SubscribeFn) {
	ctrl.subscribeMutex.Lock()
	defer ctrl.subscribeMutex.Unlock()
	ctrl.subscribeFuncs = append(ctrl.subscribeFuncs, fn)
}

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
