package inprocess

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/google/uuid"
)

// this is global so all schedulers share the same bus
var globalInProcessTransports = []*InProcessTransport{}

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
	globalInProcessTransports = append(globalInProcessTransports, inprocessTransport)
	return inprocessTransport, nil
}

/*

  PUBLIC INTERFACE

*/

func (scheduler *InProcessTransport) HostId() (string, error) {
	return scheduler.Id, nil
}

func (scheduler *InProcessTransport) Start() error {
	return nil
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (scheduler *InProcessTransport) List() (types.ListResponse, error) {
	return scheduler.GenericTransport.List()
}

func (scheduler *InProcessTransport) Get(id string) (*types.Job, error) {
	return scheduler.GenericTransport.Get(id)
}

func (scheduler *InProcessTransport) Subscribe(subscribeFunc func(jobEvent *types.JobEvent, job *types.Job)) {
	scheduler.GenericTransport.Subscribe(subscribeFunc)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER
/////////////////////////////////////////////////////////////

func (scheduler *InProcessTransport) SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error) {
	return scheduler.GenericTransport.SubmitJob(spec, deal)
}

func (scheduler *InProcessTransport) UpdateDeal(jobId string, deal *types.JobDeal) error {
	return scheduler.GenericTransport.UpdateDeal(jobId, deal)
}

func (scheduler *InProcessTransport) CancelJob(jobId string) error {
	return nil
}

func (scheduler *InProcessTransport) AcceptJobBid(jobId, nodeId string) error {
	return scheduler.GenericTransport.AcceptJobBid(jobId, nodeId)
}

func (scheduler *InProcessTransport) RejectJobBid(jobId, nodeId, message string) error {
	return scheduler.GenericTransport.RejectJobBid(jobId, nodeId, message)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (scheduler *InProcessTransport) BidJob(jobId string) error {
	return scheduler.GenericTransport.BidJob(jobId)
}

func (scheduler *InProcessTransport) SubmitResult(jobId, status, resultsId string) error {
	return scheduler.GenericTransport.SubmitResult(jobId, status, resultsId)
}

func (scheduler *InProcessTransport) ErrorJob(jobId, status string) error {
	return scheduler.GenericTransport.ErrorJob(jobId, status)
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (scheduler *InProcessTransport) ErrorJobForNode(jobId, nodeId, status string) error {
	return scheduler.GenericTransport.ErrorJobForNode(jobId, nodeId, status)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (scheduler *InProcessTransport) Connect(peerConnect string) error {
	return nil
}

// loop over all inprocess schdulers and call readJobEvent for each of them
// do this in a go-routine to simulate the network
func (scheduler *InProcessTransport) writeJobEvent(event *types.JobEvent) error {
	scheduler.Events = append(scheduler.Events, event)
	for _, loopGlobalInProcessTransport := range globalInProcessTransports {
		go func(globalInProcessTransport *InProcessTransport) {
			globalInProcessTransport.GenericTransport.ReadEvent(event)
		}(loopGlobalInProcessTransport)
	}
	return nil
}
