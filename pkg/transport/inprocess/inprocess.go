package inprocess

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/google/uuid"
)

type InProcessTransport struct {
	Ctx    context.Context
	Id     string
	Events []*types.JobEvent
	// the list of functions to call when we get an update about a job
	SubscribeFuncs []func(jobEvent *types.JobEvent, job *types.Job)

	// the writer we emit events through
	GenericTransport *transport.GenericTransport
}

func NewInprocessTransport(
	ctx context.Context,
) (*InProcessTransport, error) {
	hostId, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("Error in creating host id. %s", err)
	}
	inprocessTransport := &InProcessTransport{
		Ctx: ctx,
		Id:  hostId.String(),
	}
	inprocessTransport.GenericTransport = transport.NewGenericTransport(
		hostId.String(),
		func(event *types.JobEvent) error {
			return inprocessTransport.writeJobEvent(event)
		},
	)
	return inprocessTransport, nil
}

/*

  PUBLIC INTERFACE

*/

func (inProcessTransport *InProcessTransport) HostId() (string, error) {
	return inProcessTransport.Id, nil
}

func (inProcessTransport *InProcessTransport) Start() error {
	return nil
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (inProcessTransport *InProcessTransport) List() (types.ListResponse, error) {
	return inProcessTransport.GenericTransport.List()
}

func (inProcessTransport *InProcessTransport) Get(id string) (*types.Job, error) {
	return inProcessTransport.GenericTransport.Get(id)
}

func (inProcessTransport *InProcessTransport) Subscribe(subscribeFunc func(jobEvent *types.JobEvent, job *types.Job)) {
	inProcessTransport.GenericTransport.Subscribe(subscribeFunc)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER
/////////////////////////////////////////////////////////////

func (inProcessTransport *InProcessTransport) SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error) {
	return inProcessTransport.GenericTransport.SubmitJob(spec, deal)
}

func (inProcessTransport *InProcessTransport) UpdateDeal(jobId string, deal *types.JobDeal) error {
	return inProcessTransport.GenericTransport.UpdateDeal(jobId, deal)
}

func (inProcessTransport *InProcessTransport) CancelJob(jobId string) error {
	return nil
}

func (inProcessTransport *InProcessTransport) AcceptJobBid(jobId, nodeId string) error {
	return inProcessTransport.GenericTransport.AcceptJobBid(jobId, nodeId)
}

func (inProcessTransport *InProcessTransport) RejectJobBid(jobId, nodeId, message string) error {
	return inProcessTransport.GenericTransport.RejectJobBid(jobId, nodeId, message)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (inProcessTransport *InProcessTransport) BidJob(jobId string) error {
	return inProcessTransport.GenericTransport.BidJob(jobId)
}

func (inProcessTransport *InProcessTransport) SubmitResult(jobId, status, resultsId string) error {
	return inProcessTransport.GenericTransport.SubmitResult(jobId, status, resultsId)
}

func (inProcessTransport *InProcessTransport) ErrorJob(jobId, status string) error {
	return inProcessTransport.GenericTransport.ErrorJob(jobId, status)
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (inProcessTransport *InProcessTransport) ErrorJobForNode(jobId, nodeId, status string) error {
	return inProcessTransport.GenericTransport.ErrorJobForNode(jobId, nodeId, status)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (inProcessTransport *InProcessTransport) Connect(peerConnect string) error {
	return nil
}

// loop over all inprocess schdulers and call readJobEvent for each of them
// do this in a go-routine to simulate the network
func (inProcessTransport *InProcessTransport) writeJobEvent(event *types.JobEvent) error {
	inProcessTransport.Events = append(inProcessTransport.Events, event)
	inProcessTransport.GenericTransport.ReadEvent(event)
	return nil
}
