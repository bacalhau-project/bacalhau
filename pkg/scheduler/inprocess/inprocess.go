package inprocess

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/scheduler"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/google/uuid"
)

// this is global so all schedulers share the same bus
var globalInProcessSchedulers = []*InProcessScheduler{}

type InProcessScheduler struct {
	Ctx context.Context
	Id  string
	// the list of functions to call when we get an update about a job
	SubscribeFuncs []func(jobEvent *types.JobEvent, job *types.Job)

	// the writer we emit events through
	GenericScheduler *scheduler.GenericScheduler
}

func NewInprocessScheduler(
	ctx context.Context,
) (*InProcessScheduler, error) {
	hostId, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("Error in creating host id. %s", err)
	}
	inprocessScheduler := &InProcessScheduler{
		Ctx: ctx,
		Id:  hostId.String(),
	}
	inprocessScheduler.GenericScheduler = scheduler.NewGenericScheduler(
		hostId.String(),
		func(event *types.JobEvent) error {
			return inprocessScheduler.writeJobEvent(event)
		},
	)
	globalInProcessSchedulers = append(globalInProcessSchedulers, inprocessScheduler)
	return inprocessScheduler, nil
}

/*

  PUBLIC INTERFACE

*/

func (scheduler *InProcessScheduler) HostId() (string, error) {
	return scheduler.Id, nil
}

func (scheduler *InProcessScheduler) Start() error {
	return nil
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (scheduler *InProcessScheduler) List() (types.ListResponse, error) {
	return scheduler.GenericScheduler.List()
}

func (scheduler *InProcessScheduler) Get(id string) (*types.Job, error) {
	return scheduler.GenericScheduler.Get(id)
}

func (scheduler *InProcessScheduler) Subscribe(subscribeFunc func(jobEvent *types.JobEvent, job *types.Job)) {
	scheduler.GenericScheduler.Subscribe(subscribeFunc)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER
/////////////////////////////////////////////////////////////

func (scheduler *InProcessScheduler) SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error) {
	return scheduler.GenericScheduler.SubmitJob(spec, deal)
}

func (scheduler *InProcessScheduler) UpdateDeal(jobId string, deal *types.JobDeal) error {
	return scheduler.GenericScheduler.UpdateDeal(jobId, deal)
}

func (scheduler *InProcessScheduler) CancelJob(jobId string) error {
	return nil
}

func (scheduler *InProcessScheduler) AcceptJobBid(jobId, nodeId string) error {
	return scheduler.GenericScheduler.AcceptJobBid(jobId, nodeId)
}

func (scheduler *InProcessScheduler) RejectJobBid(jobId, nodeId, message string) error {
	return scheduler.GenericScheduler.RejectJobBid(jobId, nodeId, message)
}

func (scheduler *InProcessScheduler) AcceptResult(jobId, nodeId string) error {
	return scheduler.GenericScheduler.AcceptResult(jobId, nodeId)
}

func (scheduler *InProcessScheduler) RejectResult(jobId, nodeId, message string) error {
	return scheduler.GenericScheduler.RejectResult(jobId, nodeId, message)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (scheduler *InProcessScheduler) BidJob(jobId string) error {
	return scheduler.GenericScheduler.BidJob(jobId)
}

func (scheduler *InProcessScheduler) SubmitResult(jobId, status string, outputs []types.StorageSpec) error {
	return scheduler.GenericScheduler.SubmitResult(jobId, status, outputs)
}

func (scheduler *InProcessScheduler) ErrorJob(jobId, status string) error {
	return scheduler.GenericScheduler.ErrorJob(jobId, status)
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (scheduler *InProcessScheduler) ErrorJobForNode(jobId, nodeId, status string) error {
	return scheduler.GenericScheduler.ErrorJobForNode(jobId, nodeId, status)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (scheduler *InProcessScheduler) Connect(peerConnect string) error {
	return nil
}

// loop over all inprocess schdulers and call readJobEvent for each of them
// do this in a go-routine to simulate the network
func (scheduler *InProcessScheduler) writeJobEvent(event *types.JobEvent) error {
	for _, loopGlobalInProcessScheduler := range globalInProcessSchedulers {
		go func(globalInProcessScheduler *InProcessScheduler) {
			globalInProcessScheduler.GenericScheduler.ReadEvent(event)
		}(loopGlobalInProcessScheduler)
	}
	return nil
}
