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

			bidAccepted, err := requesterNode.ConsiderBid(job, jobEvent.NodeId)

			if err != nil {
				fmt.Printf("there was an error considering bid: %s\n", err)
				return
			}

			if bidAccepted {
				scheduler.AcceptJobBid(jobEvent.JobId, jobEvent.NodeId)
			} else {
				scheduler.RejectJobBid(jobEvent.JobId, jobEvent.NodeId, "")
			}
		}

	})

	return requesterNode, nil
}

// a compute node has bid on the job
// should we accept the bid or not?
func (node *RequesterNode) ConsiderBid(job *types.Job, nodeId string) (bool, error) {

	concurrency := job.Deal.Concurrency

	// we are already over-subscribed
	if len(job.Deal.AssignedNodes) >= concurrency {
		return false, nil
	}

	// sanity check to not allow a node to bid on a job twice
	alreadyAssigned := false

	for _, assignedNode := range job.Deal.AssignedNodes {
		if assignedNode == nodeId {
			alreadyAssigned = true
		}
	}

	if alreadyAssigned {
		return false, nil
	}

	// TODO: call out to the reputation system to decide if we want this
	// compute node to join our fleet

	return true, nil
}
