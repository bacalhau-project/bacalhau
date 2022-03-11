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

// a compute node has submitted some results
// let's check if we have >= concurrency results in the set
// if we do - then let's see which results can be grouped as the "same"
// if we have a majority in that case - let's mark those results as "accepted"
// (and reject the rest)
func (node *RequesterNode) ProcessResults(job *types.Job, nodeId string) error {

	// before we do anything - let's fetch the results for the given job
	resultsList, err := system.ProcessJobIntoResults(job)

	if err != nil {
		return err
	}

	for _, result := range *resultsList {
		err = system.FetchJobResult(result)
		if err != nil {
			return err
		}
	}

	// ok the results for this job should now be local
	// let's loop over the "AssignedNodes" and see if we have results for all of them
	// if yes - then we run the analysis on the results
	completedNodes := 0
	for _, assignedNode := range job.Deal.AssignedNodes {
		if job.State[assignedNode].State == system.JOB_STATE_COMPLETE {
			completedNodes = completedNodes + 1
		}
	}

	if completedNodes < job.Deal.Concurrency {
		return nil
	}

	// ok all of the nodes that have been assigned have marked the status as complete
	// let's work out who is "correct" and who is "incorrect"

	return nil
}
