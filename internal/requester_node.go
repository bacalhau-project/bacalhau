package internal

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/scheduler"
	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
)

type RequesterNode struct {
	Ctx       context.Context
	Scheduler scheduler.Scheduler
}

func NewRequesterNode(
	ctx context.Context,
	scheduler scheduler.Scheduler,
) (*RequesterNode, error) {

	nodeId, err := scheduler.HostId()

	if err != nil {
		return nil, err
	}

	requesterNode := &RequesterNode{
		Ctx:       ctx,
		Scheduler: scheduler,
	}

	scheduler.Subscribe(func(jobEvent *types.JobEvent, job *types.Job) {

		// we only care about jobs that we own
		if job.Owner != nodeId {
			return
		}

		switch jobEvent.EventName {

		// a compute node has bid on a job
		// let's decide if we want to accept it or not
		// we would call out to the reputation system
		// we also pay attention to the job deal concurrency setting
		case system.JOB_EVENT_BID:

			bidAccepted, message, err := requesterNode.ConsiderBid(job, jobEvent.NodeId)

			if err != nil {
				fmt.Printf("there was an error considering bid: %s\n", err)
				return
			}

			if bidAccepted {
				scheduler.AcceptJobBid(jobEvent.JobId, jobEvent.NodeId)
			} else {
				scheduler.RejectJobBid(jobEvent.JobId, jobEvent.NodeId, message)
			}

		// a compute node has submitted some results
		// let's consult our confidence and tolerance settings
		// to see if we can "accept" these results
		// or if we need to wait for some more results to arrive
		case system.JOB_EVENT_RESULTS:
			err := requesterNode.ProcessResults(job, jobEvent.NodeId)

			if err != nil {
				scheduler.ErrorJobForNode(jobEvent.JobId, jobEvent.NodeId, err.Error())
			}
		}

	})

	return requesterNode, nil
}

// a compute node has bid on the job
// should we accept the bid or not?
func (node *RequesterNode) ConsiderBid(job *types.Job, nodeId string) (bool, string, error) {

	concurrency := job.Deal.Concurrency

	// we are already over-subscribed
	if len(job.Deal.AssignedNodes) >= concurrency {
		return false, "job is over subscribed", nil
	}

	// sanity check to not allow a node to bid on a job twice
	alreadyAssigned := false

	for _, assignedNode := range job.Deal.AssignedNodes {
		if assignedNode == nodeId {
			alreadyAssigned = true
		}
	}

	if alreadyAssigned {
		return false, "this node is already assigned", nil
	}

	// TODO: call out to the reputation system to decide if we want this
	// compute node to join our fleet

	return true, "", nil
}

// some results have arrived from a compute node
// let's run over all results we currently have and compare them using the "tolerance" setting
// then let's see how many of the results we can get to agree and check the "confidence" setting
// if "number of agreeing nodes" > "confidence" - we can trigger
// "ResultsAccepted" and "ResultsRejected" methods on the scheduler interface
// we need to wait until we have at least "N >= confidence" results otherwise we have nothing
// to compare
func (node *RequesterNode) ProcessResults(job *types.Job, nodeId string) error {

	// loop over current job states
	// filter down into the ones that are "complete"
	// using the threshold - group into results that are the "same"
	// identify if there is a group with > "confidence" members
	// if yes - trigger "ResultsAccepted" and "ResultsRejected" methods on the scheduler interface
	//
	return nil
}
