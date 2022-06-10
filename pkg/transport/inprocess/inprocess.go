package inprocess

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/google/uuid"
)

// Transport is a communication channel that operates entirely in-memory, for
// testing purposes. Should not be used in production.
type Transport struct {
	id               string
	genericTransport *transport.GenericTransport

	// Public for testing purposes:
	Events []*types.JobEvent
}

func NewInprocessTransport() (*Transport, error) {
	hostID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("inprocess: error creating host.id: %w", err)
	}

	res := &Transport{
		id: hostID.String(),
	}

	res.genericTransport = transport.NewGenericTransport(
		hostID.String(),
		func(ctx context.Context, event *types.JobEvent) error {
			return res.writeJobEvent(ctx, event)
		},
	)

	return res, nil
}

/////////////////////////////////////////////////////////////
/// LIFECYCLE
/////////////////////////////////////////////////////////////

func (t *Transport) Start(ctx context.Context) error {
	<-ctx.Done() // TODO(guy): shouldn't we do something here?
	return nil
}

func (t *Transport) HostID(ctx context.Context) (string, error) {
	return t.id, nil
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (t *Transport) List(ctx context.Context) (types.ListResponse, error) {
	return t.genericTransport.List(ctx)
}

func (t *Transport) Get(ctx context.Context, id string) (*types.Job, error) {
	return t.genericTransport.Get(ctx, id)
}

func (t *Transport) Subscribe(ctx context.Context, fn func(
	jobEvent *types.JobEvent, job *types.Job)) {

	t.genericTransport.Subscribe(ctx, fn)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER NODE
/////////////////////////////////////////////////////////////

func (t *Transport) SubmitJob(ctx context.Context, spec *types.JobSpec,
	deal *types.JobDeal) (*types.Job, error) {

	return t.genericTransport.SubmitJob(ctx, spec, deal)
}

func (t *Transport) UpdateDeal(ctx context.Context, jobID string,
	deal *types.JobDeal) error {

	return t.genericTransport.UpdateDeal(ctx, jobID, deal)
}

func (t *Transport) CancelJob(ctx context.Context, jobID string) error {
	return nil
}

func (t *Transport) AcceptJobBid(ctx context.Context, jobID,
	nodeID string) error {

	return t.genericTransport.AcceptJobBid(ctx, jobID, nodeID)
}

func (t *Transport) RejectJobBid(ctx context.Context, jobID, nodeID,
	message string) error {

	return t.genericTransport.RejectJobBid(ctx, jobID, nodeID, message)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (t *Transport) BidJob(ctx context.Context, jobID string) error {
	return t.genericTransport.BidJob(ctx, jobID)
}

func (t *Transport) SubmitResult(ctx context.Context, jobID, status,
	resultsID string) error {

	return t.genericTransport.SubmitResult(ctx, jobID, status, resultsID)
}

func (t *Transport) ErrorJob(ctx context.Context, jobID, status string) error {
	return t.genericTransport.ErrorJob(ctx, jobID, status)
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (t *Transport) ErrorJobForNode(ctx context.Context, jobID, nodeID,
	status string) error {

	return t.genericTransport.ErrorJobForNode(ctx, jobID, nodeID, status)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (t *Transport) Connect(ctx context.Context, peerConnect string) error {
	return nil
}

// loop over all inprocess schdulers and call readJobEvent for each of them
// do this in a go-routine to simulate the network
func (t *Transport) writeJobEvent(ctx context.Context,
	event *types.JobEvent) error {

	t.Events = append(t.Events, event)
	t.genericTransport.BroadcastEvent(event)

	return nil
}

// Static check to ensure that Transport implements Transport:
var _ transport.Transport = (*Transport)(nil)
