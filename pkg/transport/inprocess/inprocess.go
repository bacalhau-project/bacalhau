package inprocess

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/google/uuid"
)

// Transport is a transport layer that operates entirely in-memory, for
// testing purposes. Should not be used in production.
type Transport struct {
	id string
	gt *transport.GenericTransport

	// Exported so that tests can inspect:
	Events []executor.JobEvent
}

/*

  lifecycle

*/

func NewInprocessTransport() (*InProcessTransport, error) {
	hostID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("inprocess: error creating host.id: %w", err)
	}
	res := &InProcessTransport{
		id:                 hostID.String(),
		subscribeFunctions: []transport.SubscribeFn{},
	}

	res.gt = transport.NewGenericTransport(
		hostID.String(),
		func(ctx context.Context, event executor.JobEvent) error {
			return res.writeJobEvent(ctx, event)
		},
	)

	return res, nil
}

func (t *InProcessTransport) Start(ctx context.Context) error {
	if len(t.subscribeFunctions) == 0 {
		panic("Programming error: no subscribe func, please call Subscribe immediately after constructing interface")
	}
	return nil
}

func (t *InProcessTransport) Shutdown(ctx context.Context) error {
	return nil
}

func (t *InProcessTransport) HostID(ctx context.Context) (string, error) {
	return t.id, nil
}

func (t *InProcessTransport) GetEvents() []executor.JobEvent {
	return t.seenEvents
}

/*

  pub / sub

func (t *Transport) Get(ctx context.Context, id string) (executor.Job, error) {
	return t.gt.Get(ctx, id)
}

func (t *Transport) Subscribe(ctx context.Context, fn transport.SubscribeFn) {
	t.gt.Subscribe(ctx, fn)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER NODE
/////////////////////////////////////////////////////////////

func (t *Transport) SubmitJob(ctx context.Context, spec executor.JobSpec, deal executor.JobDeal) (executor.Job, error) {
	return t.gt.SubmitJob(ctx, spec, deal)
}

func (t *Transport) UpdateDeal(ctx context.Context, jobID string, deal executor.JobDeal) error {
	return t.gt.UpdateDeal(ctx, jobID, deal)
}

func (t *Transport) CancelJob(ctx context.Context, jobID string) error {
	return nil
}

func (t *InProcessTransport) Subscribe(fn transport.SubscribeFn) {
	t.ctx.Lock()
	defer t.ctx.Unlock()
	t.subscribeFunctions = append(t.subscribeFunctions, fn)
}

func (t *Transport) RejectJobBid(ctx context.Context, jobID, nodeID, message string) error {
	return t.gt.RejectJobBid(ctx, jobID, nodeID, message)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (t *Transport) BidJob(ctx context.Context, jobID string) error {
	return t.gt.BidJob(ctx, jobID)
}

func (t *Transport) SubmitResult(ctx context.Context, jobID, status, resultsID string) error {
	return t.gt.SubmitResult(ctx, jobID, status, resultsID)
}

func (t *Transport) ErrorJob(ctx context.Context, jobID, status string) error {
	return t.gt.ErrorJob(ctx, jobID, status)
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (t *Transport) ErrorJobForNode(ctx context.Context, jobID, nodeID, status string) error {
	return t.gt.ErrorJobForNode(ctx, jobID, nodeID, status)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (t *Transport) Connect(ctx context.Context, peerConnect string) error {
	return nil
}

// loop over all inprocess schdulers and call readJobEvent for each of them
// do this in a go-routine to simulate the network
func (t *Transport) writeJobEvent(ctx context.Context, event executor.JobEvent) error {
	t.Events = append(t.Events, event)
	// async so that our stack doesn't hold onto mutexes
	go t.gt.ReadEvent(ctx, event)

	return nil
}

// Static check to ensure that Transport implements Transport:
var _ transport.Transport = (*Transport)(nil)
